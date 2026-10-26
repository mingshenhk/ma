import React, { useEffect, useRef, useState } from "react";
import { createRoot } from "react-dom/client";
import "./style.css";
import TeaHouse from "./TeaHouse";

type User = { id: string; username?: string; name: string; avatar: number };
type Config = {
  game: string;
  base: number;
  rounds: number;
  options: Record<string, number>;
};
type Def = {
  key: string;
  name: string;
  players: number;
  id: number;
  description: string;
};
type Action = { kind: string; cards?: number[]; value?: number; label: string };
type Player = {
  userId: string;
  name: string;
  avatar: number;
  ready: boolean;
  total: number;
};
type Meld = { kind: string; cards: number[]; hidden?: boolean };
type Seat = {
  hand: number[];
  count: number;
  melds: Meld[];
  score: number;
  won: boolean;
  report: number;
  missing: number;
  float: number;
};
type Win = {
  seat: number;
  from: number;
  fan: number;
  points: number;
  reasons: string[];
  hand: number[];
};
type Round = {
  number: number;
  end: string;
  scores: number[];
  wins: Win[];
  ledger: { from: number; to: number; amount: number; reason: string }[];
  at: number;
};
type Room = {
  id: string;
  owner: string;
  clubId: string;
  config: Config;
  players: (Player | null)[];
  status: string;
  version: number;
  round: number;
  me: number;
  history: Round[];
  votes: Record<string, boolean> | null;
  voteDeadline: number;
  legal?: Action[];
  state?: {
    seats: Seat[];
    phase: string;
    turn: number;
    dealer: number;
    wallCount: number;
    discards: number[][];
    table: number[];
    tableSeat: number;
    offer?: { card: number; from: number; source: string };
    end: string;
    events: { seat: number; kind: string; text: string }[];
    wins: Win[];
  };
};
type Club = {
  id: string;
  name: string;
  owner: string;
  members: Record<string, boolean>;
  managers?: Record<string, boolean>;
  joinClosed: boolean;
};
const ruleFiles: Record<string, string> = {
  hongzhong: "hongzhong",
  nohongzhong: "nohongzhong",
  hongershi: "hongershi",
  runfast: "paodekuai",
  doushisi: "doushisi",
  sichuan: "sichuan",
};
const ruleFileName = (gameKey: string) => ruleFiles[gameKey] ?? gameKey;
const avatars = ["松", "竹", "梅", "兰", "荷", "菊", "鹿", "鹤"];
const gameIcons: Record<string, string> = {
  hongzhong: "中",
  nohongzhong: "筒",
  hongershi: "贰",
  runfast: "A",
  doushisi: "⠿",
  sichuan: "萬",
};
const names: Record<string, string> = {
  waiting: "等待入座",
  playing: "对局中",
  roundEnd: "本局结束",
  finished: "房间结算",
  closed: "房间已解散",
  discard: "请出牌",
  offer: "等待响应",
  draw: "请摸牌",
  missing: "请选择定缺",
  float: "请选择漂分",
  play: "请出牌",
  prepare: "起手检查",
  declare: "请选择报胡或继续",
  flip: "请翻牌",
  respond: "等待响应",
  ended: "本局结束",
};
async function api<T = any>(
  path: string,
  body?: unknown,
  method?: string,
): Promise<T> {
  const r = await fetch("/api/" + path, {
    method: method || (body ? "POST" : "GET"),
    credentials: "same-origin",
    headers: body ? { "Content-Type": "application/json" } : undefined,
    body: body ? JSON.stringify(body) : undefined,
  });
  const data = await r.json();
  if (!r.ok) {
    const e = new Error(data.error || "请求失败") as Error & { room?: Room };
    e.room = data.room;
    throw e;
  }
  return data;
}
const uid = () =>
  typeof crypto.randomUUID === "function"
    ? crypto.randomUUID()
    : Array.from(crypto.getRandomValues(new Uint8Array(16)), (n) =>
        n.toString(16).padStart(2, "0"),
      ).join("");
function sameCards(a: number[] = [], b: number[] = []) {
  return (
    a.length === b.length &&
    [...a].sort((x, y) => x - y).join(",") ===
      [...b].sort((x, y) => x - y).join(",")
  );
}
function cardName(value: number, game: string): string {
  if (!value) return "牌背";
  if (game === "doushisi") {
    const part = (v: number) => (v > 16 ? "红" : "") + (v & 15);
    return part(value >> 8) + "点·" + part(value & 255) + "点";
  }
  if (game === "hongershi" || game === "runfast") {
    if (value === 0x4f) return "金牌";
    return (
      ["梅花", "方块", "红桃", "黑桃"][value >> 4] +
      ["", "A", "2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K"][
        value & 15
      ]
    );
  }
  if (game === "hongzhong" && value === 0x35) return "红中";
  return (value & 15) + ["", "筒", "条", "万"][value >> 4];
}
function Card({
  value,
  game,
  small = false,
  selected = false,
  onClick,
  onDoubleClick,
}: {
  value: number;
  game: string;
  small?: boolean;
  selected?: boolean;
  onClick?: () => void;
  onDoubleClick?: () => void;
}) {
  let content: React.ReactNode;
  let red = false;
  if (!value) content = <span className="back-pattern">◇</span>;
  else if (game === "doushisi") {
    const ends = [value >> 8, value & 255];
    content = (
      <span className="bone">
        {ends.map((v, i) => {
          const n = v & 15;
          return (
            <span className={"bone-end " + (v > 16 ? "red" : "")} key={i}>
              {Array.from({ length: 9 }, (_, j) => (
                <i
                  key={j}
                  className={dotPlaces[n]?.includes(j) ? "dot visible" : "dot"}
                />
              ))}
            </span>
          );
        })}
      </span>
    );
  } else if (game === "runfast" || game === "hongershi") {
    if (value === 0x4f) {
      red = true;
      content = (
        <>
          <b>王</b>
          <span className="suit">✦</span>
        </>
      );
    } else {
      const suit = value >> 4,
        rank = value & 15;
      red = suit === 1 || suit === 2;
      content = (
        <>
          <b>
            {
              [
                "",
                "A",
                "2",
                "3",
                "4",
                "5",
                "6",
                "7",
                "8",
                "9",
                "10",
                "J",
                "Q",
                "K",
              ][rank]
            }
          </b>
          <span className="suit">{["♣", "♦", "♥", "♠"][suit]}</span>
        </>
      );
    }
  } else if (game === "hongzhong" && value === 0x35) {
    red = true;
    content = <strong className="character">中</strong>;
  } else {
    const suit = value >> 4,
      rank = value & 15;
    red = suit === 3;
    content = (
      <>
        <b>{rank}</b>
        <span className={"tile-suit suit-" + suit}>
          {["", "筒", "条", "萬"][suit]}
        </span>
      </>
    );
  }
  return (
    <button
      type="button"
      aria-label={cardName(value, game)}
      onClick={onClick}
      onDoubleClick={onDoubleClick}
      className={
        "card " +
        (small ? "small " : "") +
        (red ? "red " : "") +
        (!value ? "back " : "") +
        (selected ? "selected " : "") +
        (!onClick ? "static" : "")
      }
    >
      {content}
    </button>
  );
}
const dotPlaces: Record<number, number[]> = {
  1: [4],
  2: [0, 8],
  3: [0, 4, 8],
  4: [0, 2, 6, 8],
  5: [0, 2, 4, 6, 8],
  6: [0, 2, 3, 5, 6, 8],
};
function Avatar({ p }: { p: { name: string; avatar: number } }) {
  return (
    <div className={"avatar a" + p.avatar}>
      {avatars[p.avatar % 8]}
      <span>{p.name.slice(0, 1)}</span>
    </div>
  );
}

function AdminView({ data, defs }: { data: any; defs: Def[] }) {
  const gameName = (key: string) =>
    defs.find((d) => d.key === key)?.name || key;
  return (
    <>
      <div className="admin-grid">
        {[
          ["账号", data.users],
          ["房间", data.rooms],
          ["在线连接", data.connections],
          ["请求", data.requests],
          ["操作", data.commands],
          ["存储错误", data.storageErrors],
        ].map(([k, v]) => (
          <div key={k}>
            <span>{k}</span>
            <strong>{v}</strong>
          </div>
        ))}
      </div>
      <p className="fine">
        存储：{data.service?.storage} · {data.service?.go} · 协程{" "}
        {data.service?.goroutines}
      </p>
      {data.lastStorageError && (
        <p role="alert">最近存储错误：{data.lastStorageError}</p>
      )}
      <h3>各玩法在线账号</h3>
      <div className="admin-grid">
        {defs.map((d) => (
          <div key={d.key}>
            <span>{d.name}</span>
            <strong>{data.onlineByGame?.[d.key] || 0}</strong>
          </div>
        ))}
      </div>
      <p className="fine">
        同一账号在同一玩法的多个连接只计一次；离开牌桌但停留大厅的账号未计入。
      </p>
      <details className="round-detail">
        <summary>账号资料（{data.userList?.length || 0}）</summary>
        {data.userList?.map((u: User) => (
          <p key={u.id}>
            {u.name} · {u.id}
          </p>
        ))}
      </details>
      <details className="round-detail">
        <summary>房间（{data.roomList?.length || 0}）</summary>
        {data.roomList?.map((r: any) => (
          <p key={r.room.id}>
            {r.room.id} · {gameName(r.room.config.game)} ·{" "}
            {names[r.room.status]} · {r.room.players}/{r.room.capacity}人
          </p>
        ))}
      </details>
      <details className="round-detail">
        <summary>亲友圈（{data.clubList?.length || 0}）</summary>
        {data.clubList?.map((c: Club) => (
          <p key={c.id}>
            {c.name} · {c.id} · {Object.keys(c.members).length}人 · 圈主
            {c.owner}
          </p>
        ))}
      </details>
      <details className="round-detail">
        <summary>已完成对局记录（{data.records?.length || 0}桌）</summary>
        {data.records?.map((h: any) => (
          <div className="history-item" key={h.room.id}>
            <strong>
              {h.room.id} · {gameName(h.room.config.game)} · {h.rounds.length}局
            </strong>
            <p>
              {h.players
                .filter(Boolean)
                .map((p: Player) => `${p.name} ${p.total}分`)
                .join(" / ")}
            </p>
          </div>
        ))}
      </details>
      <h3>用户反馈</h3>
      {data.reports?.length ? (
        data.reports.map((r: any) => (
          <div className="history-item" key={r.id}>
            <strong>
              反馈 {r.id} · {r.userId}
            </strong>
            <p>{r.text}</p>
          </div>
        ))
      ) : (
        <p className="muted">暂无反馈。</p>
      )}
    </>
  );
}
function App() {
  const [user, setUser] = useState<User | null>(null),
    [boot, setBoot] = useState(true),
    [defs, setDefs] = useState<Def[]>([]),
    [room, setRoom] = useState<Room | null>(null),
    [panel, setPanel] = useState(""),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false),
    [connection, setConnection] = useState("离线"),
    [selected, setSelected] = useState<number[]>([]),
    [history, setHistory] = useState<any[]>([]),
    [clubs, setClubs] = useState<Club[]>([]),
    [rules, setRules] = useState("正在读取规则…"),
    [ruleGame, setRuleGame] = useState<string | null>(null),
    [authMode, setAuthMode] = useState<"login" | "register">("login"),
    [form, setForm] = useState<Record<string, string>>({}),
    [list, setList] = useState<any[]>([]),
    [admin, setAdmin] = useState<any>(null),
    [sound, setSound] = useState(localStorage.getItem("hjy_sound") !== "0");
  const [config, setConfig] = useState<Config>({
    game: "hongzhong",
    base: 1,
    rounds: 8,
    options: {},
  });
  const ws = useRef<WebSocket | null>(null),
    roomRef = useRef<Room | null>(null),
    soundRef = useRef(sound),
    audioRef = useRef<AudioContext | null>(null),
    autoPassKey = useRef(""),
    gameList = useRef<HTMLDivElement | null>(null),
    ruleRequestId = useRef(0),
    ruleDetailOrigin = useRef("");
  roomRef.current = room;
  soundRef.current = sound;
  const flash = (s: string) => {
    setError(s);
  };
  async function copyText(text: string, success: string, fallback: string) {
    try {
      await navigator.clipboard.writeText(text);
      flash(success);
    } catch {
      flash(fallback);
    }
  }
  const accept = (next: Room) => {
    setRoom((prev) =>
      !prev || prev.id !== next.id || next.version >= prev.version
        ? next
        : prev,
    );
  };
  async function work(fn: () => Promise<void>) {
    setBusy(true);
    try {
      setError("");
      await fn();
    } catch (e) {
      const err = e as Error & { room?: Room };
      if (err.room) accept(err.room);
      flash(err.message);
    } finally {
      setBusy(false);
    }
  }
  function beep() {
    if (!soundRef.current) return;
    try {
      const ctx = audioRef.current;
      if (!ctx || ctx.state !== "running") return;
      const o = ctx.createOscillator(),
        g = ctx.createGain();
      o.frequency.value = 600;
      g.gain.setValueAtTime(0.025, ctx.currentTime);
      g.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + 0.12);
      o.connect(g);
      g.connect(ctx.destination);
      o.start();
      o.stop(ctx.currentTime + 0.12);
    } catch {}
  }
  function unlockSound() {
    try {
      audioRef.current ??= new AudioContext();
      void audioRef.current.resume();
    } catch {}
  }
  useEffect(() => {
    void api<Def[]>("games")
      .then(setDefs)
      .catch((e) => flash(e.message));
    void api("session")
      .then(async (v) => {
        setUser(v.user);
        const rid = new URLSearchParams(location.search).get("room");
        if (v.roomId) {
          accept(await api("room?id=" + v.roomId));
        } else if (rid) {
          setForm((f) => ({ ...f, join: rid }));
          setPanel("join");
        }
      })
      .catch(() => {})
      .finally(() => setBoot(false));
  }, []);
  useEffect(() => {
    if (!error) return;
    const t = setTimeout(() => setError(""), 7000);
    return () => clearTimeout(t);
  }, [error]);
  useEffect(() => {
    setSelected([]);
  }, [room?.version]);
  useEffect(() => {
    if (!room?.id || !user) return;
    let dead = false,
      retry: ReturnType<typeof setTimeout>;
    const rid = room.id;
    function connect() {
      if (dead) return;
      setConnection("连接中");
      const socket = new WebSocket(
        `${location.protocol === "https:" ? "wss" : "ws"}://${location.host}/ws?room=${rid}`,
      );
      ws.current = socket;
      socket.onopen = () => setConnection("已连接");
      socket.onmessage = (e) => {
        try {
          const msg = JSON.parse(e.data);
          if (msg.room) accept(msg.room);
          if (msg.error) flash(msg.error);
          if (msg.type === "room") beep();
        } catch {}
      };
      socket.onclose = () => {
        if (dead) return;
        setConnection("重连中");
        retry = setTimeout(connect, 1800);
      };
      socket.onerror = () => socket.close();
    }
    connect();
    return () => {
      dead = true;
      clearTimeout(retry);
      ws.current?.close();
      ws.current = null;
    };
  }, [room?.id, user?.id]);
  const field = (
    key: string,
    placeholder: string,
    type = "text",
    attrs: Omit<
      React.InputHTMLAttributes<HTMLInputElement>,
      "aria-label" | "onChange" | "placeholder" | "type" | "value"
    > = {},
  ) => (
    <input
      {...attrs}
      aria-label={placeholder}
      type={type}
      placeholder={placeholder}
      value={form[key] || ""}
      onChange={(e) => setForm({ ...form, [key]: e.target.value })}
    />
  );
  const command = async (op: string, extra: Record<string, unknown> = {}) => {
    const r = roomRef.current;
    const c = {
      op,
      roomId: r?.id,
      version: r?.version,
      requestId: uid(),
      ...extra,
    };
    const value = await api<Room>("command", c);
    if (op === "leave") {
      setRoom(null);
    } else accept(value);
  };
  const act = (a: Action) => {
    unlockSound();
    void work(async () => {
      const r = roomRef.current;
      if (!r) return;
      const cmd = {
        op: "action",
        roomId: r.id,
        version: r.version,
        requestId: uid(),
        action: a,
      };
      if (ws.current?.readyState === WebSocket.OPEN) {
        ws.current.send(JSON.stringify(cmd));
      } else {
        accept(await api<Room>("command", cmd));
      }
    });
  };
  async function openPanel(p: string) {
    setPanel(p);
    setError("");
    try {
      if (p === "history") setHistory((await api("history")) || []);
      if (p === "clubs" || p === "create") setClubs((await api("clubs")) || []);
      if (p === "join") setList((await api("rooms")) || []);
    } catch (e) {
      flash((e as Error).message);
    }
  }
  async function openRules(gameKey: string) {
    const isKnownGame = defs.some((game) => game.key === gameKey);
    const isMatchingCreatePanel = panel === "create" && config.game === gameKey;
    const isMatchingRoom = room?.config.game === gameKey;
    if (!isKnownGame || (!isMatchingCreatePanel && !isMatchingRoom)) return;

    ruleDetailOrigin.current = panel;
    const requestId = ++ruleRequestId.current;
    setPanel("rules");
    setError("");
    setRuleGame(gameKey);
    setRules("正在读取本游戏规则…");
    try {
      const response = await fetch(
        `/rules/${encodeURIComponent(ruleFileName(gameKey))}.txt`,
      );
      if (!response.ok) throw new Error("暂时无法读取这款游戏的规则。");
      const text = await response.text();
      if (requestId === ruleRequestId.current) setRules(text);
    } catch (e) {
      if (requestId === ruleRequestId.current) {
        setRules("规则加载失败，请稍后重试。");
        flash((e as Error).message);
      }
    }
  }
  function backFromRules() {
    ++ruleRequestId.current;
    const origin = ruleDetailOrigin.current;
    setRuleGame(null);
    setPanel(origin);
  }
  function closeRules() {
    ++ruleRequestId.current;
    const origin = ruleDetailOrigin.current;
    setRuleGame(null);
    setPanel(origin);
  }
  const def = defs.find((d) => d.key === config.game),
    roomDef = defs.find((d) => d.key === room?.config.game),
    my = room?.players[room.me],
    isOwner = room?.owner === user?.id;
  const selectedCards = selected.map(
    (i) => room?.state?.seats[room.me]?.hand[i] || 0,
  );
  const actions = room?.legal || [];
  const isMahjong = ["hongzhong", "nohongzhong", "sichuan"].includes(
    room?.config.game || "",
  );
  const isCardGame = ["runfast", "hongershi", "doushisi"].includes(
    room?.config.game || "",
  );
  const autoPass =
    room?.status === "playing" &&
    room.state?.phase !== "missing" &&
    actions.length === 1 &&
    actions[0]?.kind === "pass";
  const direct = actions.filter(
    (a) => !a.cards?.length && !(autoPass && a.kind === "pass"),
  );
  const cardActions = actions.filter(
    (a) =>
      a.cards?.length &&
      (!selected.length || sameCards(a.cards, selectedCards)),
  );
  const selectedPlayAction = (() => {
    if (!isCardGame || !selected.length) return undefined;
    const legal = actions.find(
      (a) => a.kind === "play" && sameCards(a.cards || [], selectedCards),
    );
    if (legal) return legal;
    return { kind: "play", cards: selectedCards, label: "出牌" };
  })();
  const visibleCardActions = cardActions.filter((a) =>
    isCardGame ? a.kind !== "play" : isMahjong ? a.kind !== "discard" : true,
  );
  useEffect(() => {
    if (!autoPass || !room) return;
    const key = `${room.id}:${room.version}`;
    if (autoPassKey.current === key) return;
    autoPassKey.current = key;
    act(actions[0]);
  }, [autoPass, room?.id, room?.version]);
  const colorClass = (n: number) =>
    n > 0 ? "positive" : n < 0 ? "negative" : "";
  function opt(key: string, value: number) {
    setConfig((c) => ({ ...c, options: { ...c.options, [key]: value } }));
  }
  const option = (
    key: string,
    title: string,
    choices: [number, string][],
    defaultValue: number,
  ) => (
    <label className="option" key={key}>
      <span>{title}</span>
      <select
        value={config.options[key] ?? defaultValue}
        onChange={(e) => opt(key, Number(e.target.value))}
      >
        {choices.map(([v, label]) => (
          <option value={v} key={v}>
            {label}
          </option>
        ))}
      </select>
    </label>
  );
  if (boot)
    return (
      <main className="loading">
        <img src="/icon.svg" />
        <h1>浩家园</h1>
        <p>正在准备茶桌…</p>
      </main>
    );
  return (
    <main
      onPointerDown={unlockSound}
      className={room ? "app in-room" : user ? "app lobby-app" : "app"}
    >
      <div className="grain" />
      <div className="scenery mountain-one" />
      <div className="scenery mountain-two" />
      {error && (
        <div className="toast" role="alert" onClick={() => setError("")}>
          {error}
          <span>×</span>
        </div>
      )}
      <header className="topbar">
        <button
          className="brand"
          onClick={() => (room ? void openPanel("tableMenu") : setPanel(""))}
        >
          <img src="/icon.svg" />
          <span>
            浩家园<small>HAO JIA YUAN</small>
          </span>
        </button>
        {user && !room && (
          <button
            className="profile top-profile"
            onClick={() => {
              setForm({
                ...form,
                profileName: user.name,
                avatar: String(user.avatar),
              });
              setPanel("profile");
            }}
          >
            <Avatar p={user} />
            <span>
              <strong>{user.name}</strong>
              <small>ID {user.id}</small>
            </span>
          </button>
        )}
        <div className="top-note">亲友相聚 · 文明娱乐</div>
        <div className="top-actions">
          {room && (
            <span
              className={
                "connection " + (connection === "已连接" ? "online" : "")
              }
            >
              ● {connection}
            </span>
          )}
          <button
            className="icon-button"
            title="设置"
            onClick={() => setPanel("settings")}
          >
            ⚙
          </button>
        </div>
      </header>
      {!user ? (
        <section className="welcome">
          <div className="welcome-copy">
            <span className="eyebrow">一桌好牌 · 一院好友</span>
            <h1>
              好时光，
              <br />
              在浩家园。
            </h1>
            <p>熟悉的家乡玩法，随时邀请亲友入座。</p>
            <div className="welcome-tiles">
              <Card value={0x35} game="hongzhong" />
              <Card value={0x18} game="hongzhong" />
              <Card value={0x28} game="hongzhong" />
            </div>
          </div>
          <form
            className="login-paper"
            onSubmit={(e) => {
              e.preventDefault();
              void work(async () => {
                const v = await api(authMode, {
                  username: form.username?.trim(),
                  password: form.password,
                });
                setUser(v.user);
                setForm((f) => ({ ...f, password: "" }));
                if (v.roomId) {
                  accept(await api("room?id=" + v.roomId));
                  setPanel("");
                }
                const invite = new URLSearchParams(location.search).get("room");
                if (!v.roomId && invite) {
                  setForm((f) => ({ ...f, join: invite }));
                  setPanel("join");
                }
              });
            }}
          >
            <span className="eyebrow dark">{authMode === "login" ? "欢迎回来" : "创建家园账号"}</span>
            <h2>{authMode === "login" ? "登录浩家园" : "注册浩家园"}</h2>
            {field("username", "用户名（3–24位）", "text", {
              autoComplete: authMode === "login" ? "username" : "username",
              minLength: 3,
              maxLength: 24,
              required: true,
            })}
            {field("password", "密码（至少8位）", "password", {
              autoComplete: authMode === "login" ? "current-password" : "new-password",
              minLength: authMode === "register" ? 8 : undefined,
              maxLength: 64,
              required: true,
            })}
            <button disabled={busy} className="primary" type="submit">
              {authMode === "login" ? "登录" : "注册并登录"} <span>→</span>
            </button>
            <button
              className="text-link"
              type="button"
              onClick={() => {
                setError("");
                setAuthMode((mode) => mode === "login" ? "register" : "login");
              }}
            >
              {authMode === "login" ? "还没有账号？立即注册" : "已有账号？返回登录"}
            </button>
            <p className="fine">
              内部试玩 · 仅使用娱乐积分
              <br />
              当前使用本项目六款正式玩法规则。
            </p>
          </form>
        </section>
      ) : !room ? (
        <>
          <section className="lobby">
            <aside className="lobby-left">
              <button
                className="join-room"
                onClick={() => void openPanel("join")}
              >
                <span className="join-glyph">⌑</span>
                <span>
                  加入房间<small>输入房号，与好友相聚</small>
                </span>
                <b>↗</b>
              </button>
              <button
                className="create-room"
                onClick={() => void openPanel("create")}
              >
                创建房间 <span>＋</span>
              </button>
              <div className="announcement">
                <span>公 告</span>
                前五款固定两人，普通麻将四人。茶馆内可保存玩法、审核成员、开桌及查看战绩。
              </div>
              <button
                className="tea-lobby-entry"
                onClick={() => void openPanel("clubs")}
              >
                <span>♧</span>
                <strong>进入茶馆</strong>
                <small>馆内玩法 · 亲友桌台</small>
                <b>›</b>
              </button>
            </aside>
            <section className="lobby-scene" aria-label="川友棋牌大厅">
              <div className="scene-lantern lantern-left" aria-hidden="true">✦</div>
              <div className="scene-lantern lantern-right" aria-hidden="true">✦</div>
              <div className="scene-window" aria-hidden="true">
                <span>川</span><span>友</span><span>相</span><span>聚</span>
              </div>
              <div className="scene-table" aria-hidden="true">
                <div className="scene-rim"><div className="scene-felt" /></div>
                <div className="scene-tile tile-center">中</div>
                <div className="scene-tile tile-right">發</div>
                <div className="scene-tile tile-left">筒</div>
                <div className="scene-tile tile-bottom">條</div>
              </div>
              <div className="scene-caption">
                <span>♧</span>
                <div><strong>川友茶席</strong><small>约上亲友 · 开桌切磋牌技</small></div>
              </div>
            </section>
            <section className="games-section">
              <div className="section-title">
                <div>
                  <h2>游戏列表</h2>
                </div>
                <span>共 6 款玩法</span>
              </div>
              <button
                className="game-scroll"
                aria-label="向上浏览游戏"
                onClick={() =>
                  gameList.current?.scrollBy({ top: -320, behavior: "smooth" })
                }
              >
                ⌃
              </button>
              <div className="games-list" ref={gameList}>
                {defs.map((d, i) => (
                  <article
                    key={d.key}
                    className={"game-tile tile-" + d.key}
                  >
                    <div className="game-meta">
                      <span>{d.players} 人</span>
                      <small>0{i + 1}</small>
                    </div>
                    <div className="game-art">
                      <span>{gameIcons[d.key]}</span>
                      <i>
                        {d.key === "runfast"
                          ? "♠"
                          : d.key === "doushisi"
                            ? "⠿"
                            : d.key === "hongershi"
                              ? "♥"
                              : "八"}
                      </i>
                    </div>
                    <h3>{d.name}</h3>
                    <p>{d.description}</p>
                    <div className="game-actions">
                      <button
                        type="button"
                        className="game-enter game-start"
                        aria-label={`创建${d.name}房间`}
                        onClick={() => {
                          setConfig({
                            game: d.key,
                            base: 1,
                            rounds:
                              d.key === "runfast"
                                ? 10
                                : d.key === "hongershi"
                                  ? 4
                                  : 8,
                            options: {},
                          });
                          void openPanel("create");
                        }}
                      >
                        <span>
                          创建房间
                        </span>
                        <b>→</b>
                      </button>
                    </div>
                  </article>
                ))}
              </div>
              <button
                className="game-scroll"
                aria-label="向下浏览游戏"
                onClick={() =>
                  gameList.current?.scrollBy({ top: 320, behavior: "smooth" })
                }
              >
                ⌄
              </button>
            </section>
          </section>
          <footer className="bottom-nav">
            <span className="footer-label">
              HAOJIAYUAN <i>·</i> 娱乐积分试玩
            </span>
            <nav>
              <button onClick={() => void openPanel("history")}>◷ 战绩</button>
              <button
                onClick={() => {
                  setForm({
                    ...form,
                    profileName: user.name,
                    avatar: String(user.avatar),
                  });
                  setPanel("profile");
                }}
              >
                ♙ 个人
              </button>
              <button onClick={() => setPanel("settings")}>⚙ 设置</button>
              <button
                className="nav-tea"
                onClick={() => void openPanel("clubs")}
              >
                ♧ 茶馆
              </button>
            </nav>
          </footer>
        </>
      ) : (
        <section className="table-shell">
          <div className="table-meta">
            <button className="quiet" onClick={() => setPanel("tableMenu")}>
              ‹ 菜单
            </button>
            <span>
              {roomDef?.name} <b>{room.players.length}人</b>
            </span>
            <span>
              房间 {room.id} · {room.round}/{room.config.rounds}局 · 底分
              {room.config.base}
            </span>
            <button
              className="quiet"
              onClick={() => {
                void copyText(
                  `${location.origin}/?room=${room.id}`,
                  "邀请链接已复制",
                  "房间号：" + room.id,
                );
              }}
            >
              邀请好友 ↗
            </button>
          </div>
          <div className="felt-table">
            <div className="table-ring" />
            <div className="table-watermark">
              <img src="/icon.svg" />
              <strong>浩家园</strong>
              <span>亲友相聚 · 闲来一局</span>
            </div>
            {room.players.map((p, i) => {
              const relative =
                (i - room.me + room.players.length) % room.players.length;
              const pos =
                room.players.length === 2
                  ? relative === 0
                    ? "self"
                    : "opposite"
                  : ["self", "right", "opposite", "left"][relative];
              const seat = room.state?.seats[i];
              return (
                <div
                  className={
                    "seat seat-" +
                    pos +
                    (room.state?.turn === i &&
                    room.status === "playing" &&
                    room.state.phase !== "missing"
                      ? " active"
                      : "")
                  }
                  key={i}
                >
                  {p ? (
                    <>
                      <div className="seat-person">
                        <Avatar p={p} />
                        <div>
                          <strong>
                            {p.name}
                            {i === room.me ? "（我）" : ""}
                          </strong>
                          <small>
                            {p.ready
                              ? "已准备"
                              : room.status === "waiting"
                                ? "未准备"
                                : "真人玩家"}
                            {room.state?.dealer === i ? " · 庄" : ""}
                          </small>
                          <b
                            className={colorClass(p.total + (seat?.score || 0))}
                          >
                            {p.total +
                              (room.status === "playing"
                                ? seat?.score || 0
                                : 0)}{" "}
                            分
                          </b>
                        </div>
                      </div>
                      {seat && i !== room.me && (
                        <div className="opponent-hand">
                          {seat.hand.length ? (
                            seat.hand.map((c, j) => (
                              <Card
                                key={j}
                                value={c}
                                game={room.config.game}
                                small
                              />
                            ))
                          ) : (
                            <>
                              <span className="mini-back">▥ ▥ ▥</span>
                              <small>
                                {seat.count}张{seat.won ? " · 已胡" : ""}
                                {seat.missing > 0
                                  ? " · 缺" +
                                    ["", "筒", "条", "万"][seat.missing]
                                  : ""}
                              </small>
                            </>
                          )}
                        </div>
                      )}
                      {seat?.melds.length ? (
                        <div className="melds">
                          {seat.melds.map((m, j) => (
                            <span className="meld" key={j}>
                              {m.cards.map((c, k) => (
                                <Card
                                  key={k}
                                  value={c}
                                  game={room.config.game}
                                  small
                                />
                              ))}
                            </span>
                          ))}
                        </div>
                      ) : null}
                    </>
                  ) : (
                    <div className="empty-seat">
                      <span>＋</span>等待好友入座
                    </div>
                  )}
                </div>
              );
            })}
            <div className="table-center">
              {!room.state ? (
                <div className="waiting-card">
                  <span className="eyebrow">房号 {room.id}</span>
                  <h2>好牌，就等你来</h2>
                  <p>邀请好友入座后即可开始对局。</p>
                </div>
              ) : (
                <>
                  <div className="turn-badge">
                    {room.status === "playing"
                      ? room.state.phase === "missing"
                        ? "请各自选择定缺"
                        : `${room.players[room.state.turn]?.name} · ${names[room.state.phase] || room.state.phase}`
                      : names[room.status]}
                    <small>
                      余牌 {room.state.wallCount} 张{" "}
                    </small>
                  </div>
                  <div className="discard-fields">
                    {room.state.discards?.map((cards, i) => (
                      <div key={i} className="discard-seat">
                        <small>{room.players[i]?.name}</small>
                        <div>
                          {(cards || []).slice(-20).map((c, j) => (
                            <Card
                              key={j}
                              value={c}
                              game={room.config.game}
                              small
                            />
                          ))}
                        </div>
                      </div>
                    ))}
                  </div>
                  {room.state.table?.length > 0 && (
                    <div className="played-cards">
                      {room.state.table.map((c, i) => (
                        <Card key={i} value={c} game={room.config.game} small />
                      ))}
                    </div>
                  )}
                  {room.state.offer && (
                    <div className="offer-card">
                      <small>
                        {room.players[room.state.offer.from]?.name} 出牌
                      </small>
                      <Card
                        value={room.state.offer.card}
                        game={room.config.game}
                        small
                      />
                    </div>
                  )}
                </>
              )}
            </div>
            {room.status !== "playing" ? (
              <div className="table-controls">
                {(room.status === "waiting" || room.status === "roundEnd") && (
                  <>
                    <button
                      className={my?.ready ? "secondary" : "primary"}
                      disabled={busy}
                      onClick={() => void work(() => command("ready"))}
                    >
                      {my?.ready ? "取消准备" : "准备"}
                    </button>
                    {isOwner && (
                      <button
                        className="primary"
                        disabled={busy}
                        onClick={() =>
                          void work(() =>
                            command(
                              room.status === "waiting" ? "start" : "next",
                            ),
                          )
                        }
                      >
                        {room.status === "waiting" ? "开始游戏" : "开始下一局"}
                      </button>
                    )}
                    {room.status === "roundEnd" && (
                      <button
                        className="secondary"
                        onClick={() => setPanel("settlement")}
                      >
                        查看结算
                      </button>
                    )}
                  </>
                )}
                {(room.status === "finished" || room.status === "closed") && (
                  <>
                    <button
                      className="primary"
                      onClick={() => setPanel("settlement")}
                    >
                      查看总战绩
                    </button>
                    <button className="secondary" onClick={() => setRoom(null)}>
                      返回大厅
                    </button>
                  </>
                )}
              </div>
            ) : (
              <div className="action-dock">
                <div className="action-tip">
                  {selected.length
                    ? `已选 ${selected.length} 张`
                    : actions.length
                      ? "轮到你了，选择合法操作"
                      : "等待其他玩家…"}{" "}
                  <button onClick={() => setSelected([])}>重选</button>
                </div>
                <div className="action-buttons">
                  {direct.map((a, i) => (
                    <button
                      disabled={busy}
                      className={a.kind === "hu" ? "win-button" : "secondary"}
                      key={"d" + i}
                      onClick={() => act(a)}
                    >
                      {a.label}
                    </button>
                  ))}
                  {visibleCardActions.slice(0, 12).map((a, i) => (
                    <button
                      disabled={busy}
                      className={a.kind === "hu" ? "win-button" : "primary"}
                      key={"c" + i}
                      onClick={() => act(a)}
                    >
                      {a.label}
                      <small>
                        {a.cards
                          ?.map((c) => cardName(c, room.config.game))
                          .join(" ")}
                      </small>
                    </button>
                  ))}
                  {visibleCardActions.length > 12 && (
                    <select
                      aria-label="更多合法组合"
                      value=""
                      onChange={(e) => {
                        if (e.target.value)
                          act(visibleCardActions[Number(e.target.value)]);
                      }}
                    >
                      <option value="">
                        其余 {visibleCardActions.length - 12} 个组合
                      </option>
                      {visibleCardActions.slice(12).map((a, i) => (
                        <option value={i + 12} key={i}>
                          {a.label} [
                          {a.cards
                            ?.map((c) => cardName(c, room.config.game))
                            .join(" ")}
                          ]
                        </option>
                      ))}
                    </select>
                  )}
                  {isCardGame && selected.length > 0 && (
                    <button
                      disabled={busy || !selectedPlayAction}
                      className="primary"
                      onClick={() => selectedPlayAction && act(selectedPlayAction)}
                    >
                      {selectedPlayAction?.label || "出牌"}
                    </button>
                  )}
                </div>
              </div>
            )}
            {room.state && (
              <div className="my-hand-area">
                <div className="hand-caption">
                  我的手牌{" "}
                  <span>
                    {room.state.phase === "float"
                      ? "漂分确定前不展示手牌"
                      : `${room.state.seats[room.me]?.count}张`}
                  </span>
                </div>
                <div className="my-hand">
                  {room.state.seats[room.me]?.hand.map((c, i) => (
                    <Card
                      key={i}
                      value={c}
                      game={room.config.game}
                      selected={selected.includes(i)}
                      onClick={
                        room.status === "playing"
                          ? () => {
                              if (isMahjong) setSelected([i]);
                              else
                                setSelected((s) =>
                                  s.includes(i)
                                    ? s.filter((v) => v !== i)
                                    : [...s, i],
                                );
                            }
                          : undefined
                      }
                      onDoubleClick={
                        isMahjong && room.status === "playing"
                          ? () => {
                              const discard = actions.find(
                                (a) =>
                                  a.kind === "discard" &&
                                  sameCards(a.cards || [], [c]),
                              );
                              if (discard) {
                                setSelected([]);
                                act(discard);
                              } else {
                                flash("现在不能打出这张牌");
                              }
                            }
                          : undefined
                      }
                    />
                  ))}
                </div>
                {room.state.seats[room.me]?.melds.length ? (
                  <div className="my-melds">
                    {room.state.seats[room.me].melds.map((m, j) => (
                      <span className="meld" key={j}>
                        <small>
                          {{
                            peng: "碰",
                            kong: "杠",
                            eat: "吃",
                            king: "金牌",
                            bonus: "花牌",
                          }[m.kind] || m.kind}
                        </small>
                        {m.cards.map((c, k) => (
                          <Card
                            key={k}
                            value={c}
                            game={room.config.game}
                            small
                          />
                        ))}
                      </span>
                    ))}
                  </div>
                ) : null}
              </div>
            )}
          </div>
          <div className="table-log">
            {room.state?.events.slice(-2).map((e, i) => (
              <span key={i}>
                {e.seat >= 0 ? room.players[e.seat]?.name + "：" : ""}
                {e.text}
              </span>
            ))}
            <button onClick={() => setPanel("settlement")}>本桌战绩 →</button>
          </div>
          {room.votes && (
            <div className="vote-banner">
              正在申请解散（{Object.keys(room.votes).length}票）
              <button onClick={() => void work(() => command("dissolve"))}>
                同意
              </button>
              <button
                onClick={() => void work(() => command("rejectDissolve"))}
              >
                继续游戏
              </button>
            </div>
          )}
        </section>
      )}
      {panel && (
        <div
          className="modal-backdrop"
          onClick={() => (panel === "rules" ? closeRules() : setPanel(""))}
        >
          <section
            className={"modal modal-" + panel}
            role="dialog"
            aria-modal="true"
            aria-label={panel}
            onClick={(e) => e.stopPropagation()}
          >
            <button
              className="close-modal"
              title="关闭"
              onClick={() => (panel === "rules" ? closeRules() : setPanel(""))}
            >
              ×
            </button>
            {panel === "create" && (
              <>
                <span className="eyebrow dark">CREATE A ROOM</span>
                <h2>开一桌，等好友</h2>
                <div className="game-picker">
                  {defs.map((d) => (
                    <div className="game-picker-item" key={d.key}>
                      <button
                        type="button"
                        className={
                          "game-picker-select" +
                          (d.key === config.game ? " chosen" : "")
                        }
                        aria-pressed={d.key === config.game}
                        onClick={() =>
                          setConfig({
                            game: d.key,
                            base: 1,
                            rounds:
                              d.key === "runfast"
                                ? 10
                                : d.key === "hongershi"
                                  ? 4
                                  : 8,
                            options: {},
                          })
                        }
                      >
                        <span>{d.name}</span>
                        <small>{d.players}人</small>
                      </button>
                    </div>
                  ))}
                </div>
                <div className="rule-note">
                  {def?.description}{" "}
                  · 本项目正式规则
                </div>
                <div className="selected-game-rules">
                  <span>已选择：{def?.name}（{def?.players}人）</span>
                  <button
                    type="button"
                    onClick={() => void openRules(config.game)}
                    aria-label={`查看${def?.name}详细规则`}
                  >
                    查看本游戏详细规则
                  </button>
                </div>
                <div className="options-grid">
                  <label className="option">
                    <span>底分</span>
                    <select
                      value={config.base}
                      onChange={(e) =>
                        setConfig({ ...config, base: Number(e.target.value) })
                      }
                    >
                      {[1, 2, 5, 10].map((v) => (
                        <option key={v}>{v}</option>
                      ))}
                    </select>
                  </label>
                  <label className="option">
                    <span>局数</span>
                    <select
                      value={config.rounds}
                      onChange={(e) =>
                        setConfig({ ...config, rounds: Number(e.target.value) })
                      }
                    >
                      {(config.game === "runfast"
                        ? [10, 20]
                        : config.game === "hongershi"
                          ? [4, 8, 12]
                          : config.game === "doushisi"
                            ? [8, 12, 16]
                            : [8, 16, 32]
                      ).map((v) => (
                        <option key={v}>{v}</option>
                      ))}
                    </select>
                  </label>
                  {["hongzhong", "nohongzhong", "sichuan"].includes(
                    config.game,
                  ) &&
                    option(
                      "root",
                      "根加番",
                      [
                        [1, "开启"],
                        [0, "关闭"],
                      ],
                      1,
                    )}
                  {["hongzhong", "nohongzhong"].includes(config.game) && (
                    <>
                      {option(
                        "ka",
                        "456卡",
                        [
                          [1, "开启"],
                          [0, "关闭"],
                        ],
                        1,
                      )}
                      {option(
                        "minFan",
                        "起胡",
                        [
                          [1, "1番"],
                          [2, "2番"],
                          [3, "3番"],
                        ],
                        1,
                      )}
                    </>
                  )}
                  {config.game === "hongershi" && (
                    <>
                      {option(
                        "kings",
                        "金牌张数",
                        [
                          [6, "6张"],
                          [9, "9张"],
                          [12, "12张"],
                        ],
                        6,
                      )}
                      {[
                        "redFive",
                        "blackFive",
                        "selfEat",
                        "ladder",
                        "checkListen",
                      ].map((k, i) =>
                        option(
                          k,
                          ["红五", "黑五", "自吃", "阶梯", "查叫"][i],
                          [
                            [1, "开启"],
                            [0, "关闭"],
                          ],
                          k === "checkListen" ? 1 : 0,
                        ),
                      )}
                      {option(
                        "cap",
                        "封顶",
                        [
                          [4, "4番"],
                          [5, "5番"],
                          [6, "6番"],
                        ],
                        4,
                      )}
                      {option(
                        "listenMode",
                        "报听方式",
                        [
                          [0, "活听"],
                          [1, "死听"],
                        ],
                        0,
                      )}
                    </>
                  )}
                  {config.game === "doushisi" && (
                    <>
                      {option(
                        "allowFloat",
                        "漂分",
                        [
                          [1, "允许"],
                          [0, "关闭"],
                        ],
                        1,
                      )}
                      {option(
                        "cap",
                        "封顶",
                        [
                          [500, "500"],
                          [1000, "1000"],
                        ],
                        500,
                      )}
                      {option(
                        "ladder",
                        "阶梯",
                        [
                          [1, "开启"],
                          [0, "关闭"],
                        ],
                        0,
                      )}
                    </>
                  )}
                  {config.game === "runfast" && (
                    <>
                      {option(
                        "random",
                        "减牌方式",
                        [
                          [0, "固定23张牌"],
                          [1, "随机40张牌"],
                        ],
                        0,
                      )}
                      {option(
                        "guan",
                        "关牌",
                        [
                          [1, "开启"],
                          [0, "关闭"],
                        ],
                        1,
                      )}
                      {option(
                        "bombCap",
                        "炸弹封顶",
                        [
                          [-1, "不封顶"],
                          [1, "1个"],
                          [3, "3个"],
                        ],
                        1,
                      )}
                    </>
                  )}
                </div>
                <button
                  className="primary full"
                  disabled={busy}
                  onClick={() =>
                    void work(async () => {
                      await command("create", { config });
                      setPanel("");
                    })
                  }
                >
                  创建 {def?.players} 人房间 →
                </button>
              </>
            )}
            {panel === "join" && (
              <>
                <span className="eyebrow dark">JOIN FRIENDS</span>
                <h2>好友在等你</h2>
                <form
                  onSubmit={(e) => {
                    e.preventDefault();
                    void work(async () => {
                      await command("join", { roomId: form.join });
                      setPanel("");
                    });
                  }}
                >
                  {field("join", "输入6位房间号", "text")}
                  <button className="primary full" disabled={busy}>
                    加入房间 →
                  </button>
                </form>
                <h3>等待中的牌桌</h3>
                <div className="room-list">
                  {list.length ? (
                    list.map((r) => (
                      <button
                        key={r.id}
                        onClick={() =>
                          void work(async () => {
                            await command("join", { roomId: r.id });
                            setPanel("");
                          })
                        }
                      >
                        <strong>
                          {defs.find((d) => d.key === r.config.game)?.name}
                        </strong>
                        <span>
                          {r.id} · {r.players}/{r.capacity}人
                        </span>
                        <b>加入 →</b>
                      </button>
                    ))
                  ) : (
                    <p className="muted">暂无等待房间，可以先创建一桌。</p>
                  )}
                </div>
              </>
            )}
            {panel === "settings" && (
              <>
                <h2>家园设置</h2>
                <label className="toggle-row">
                  操作提示音
                  <input
                    type="checkbox"
                    checked={sound}
                    onChange={(e) => {
                      setSound(e.target.checked);
                      localStorage.setItem(
                        "hjy_sound",
                        e.target.checked ? "1" : "0",
                      );
                      unlockSound();
                    }}
                  />
                </label>
                <button
                  className="secondary full"
                  onClick={() => {
                    void document.documentElement
                      .requestFullscreen?.()
                      .catch(() => flash("当前浏览器不支持全屏"));
                  }}
                >
                  进入全屏
                </button>
                <p className="fine">
                  横屏体验更佳。网络恢复后会自动重连；只有过牌可选时才会自动过牌，有碰、杠、胡等操作时由玩家选择。出牌由玩家自己决定，系统只校验规则是否合法。
                </p>
                <p className="fine">
                  添加到桌面：iPhone Safari 使用“分享 → 添加到主屏幕”；Android
                  浏览器使用菜单中的“添加到主屏幕／安装”。微信内请先用系统浏览器打开。公网安装建议使用
                  HTTPS；此版本需联网使用。
                </p>
                <button className="text-link" onClick={() => setPanel("admin")}>
                  管理入口
                </button>
                {user && !room && (
                  <button
                    className="danger full"
                    onClick={() =>
                      void work(async () => {
                        await api("session", undefined, "DELETE");
                        setUser(null);
                        setPanel("");
                      })
                    }
                  >
                    退出当前账号
                  </button>
                )}
              </>
            )}
            {panel === "profile" && user && (
              <>
                <h2>我的资料</h2>
                <p className="muted">ID {user.id}</p>
                {field("profileName", "昵称")}
                <div className="avatar-picker">
                  {avatars.map((a, i) => (
                    <button
                      className={Number(form.avatar) === i ? "chosen" : ""}
                      key={i}
                      onClick={() => setForm({ ...form, avatar: String(i) })}
                    >
                      {a}
                    </button>
                  ))}
                </div>
                <button
                  className="primary full"
                  onClick={() =>
                    void work(async () => {
                      setUser(
                        await api("profile", {
                          name: form.profileName,
                          avatar: Number(form.avatar),
                        }),
                      );
                      setPanel("");
                    })
                  }
                >
                  保存资料
                </button>
                <p className="fine">
                  使用用户名和密码登录。用户名用于登录，昵称可在资料中单独修改。
                </p>
              </>
            )}
            {panel === "rules" && (
              <>
                <span className="eyebrow dark">HOUSE RULES</span>
                <button
                  className="rules-back"
                  type="button"
                  onClick={backFromRules}
                >
                  ← 返回上一步
                </button>
                <h2>{defs.find((d) => d.key === ruleGame)?.name} · 详细规则</h2>
                <p className="muted">
                  {defs.find((d) => d.key === ruleGame)?.players} 人玩法 ·
                  本项目正式规则
                </p>
                <a
                  className="text-link"
                  href={`/rules/${encodeURIComponent(ruleFileName(ruleGame ?? ""))}.txt`}
                  download
                >
                  下载本游戏规则 TXT ↗
                </a>
                <pre className="rules-text">{rules}</pre>
              </>
            )}
            {panel === "history" && (
              <>
                <h2>我的战绩</h2>
                {history.length ? (
                  history.map((h) => (
                    <div className="history-item" key={h.room.id}>
                      <strong>
                        {defs.find((d) => d.key === h.room.config.game)?.name} ·
                        房间{h.room.id}
                      </strong>
                      <small>
                        {new Date(h.updated * 1000).toLocaleString()}
                      </small>
                      <p>
                        {h.players
                          .filter(Boolean)
                          .map((p: Player) => `${p.name} ${p.total}分`)
                          .join(" / ")}
                      </p>
                      <span>
                        已完成 {h.rounds.length} 局 · {names[h.room.status]}
                      </span>
                    </div>
                  ))
                ) : (
                  <p className="muted">完成第一局后，战绩会保存在这里。</p>
                )}
              </>
            )}
            {panel === "settlement" && room && (
              <>
                <h2>
                  {room.status === "finished" ? "房间总战绩" : "本桌战绩"}
                </h2>
                <div className="scoreboard">
                  {room.players.map(
                    (p, i) =>
                      p && (
                        <div key={i}>
                          <Avatar p={p} />
                          <strong>{p.name}</strong>
                          <b className={colorClass(p.total)}>
                            {p.total > 0 ? "+" : ""}
                            {p.total}
                          </b>
                        </div>
                      ),
                  )}
                </div>
                {room.history?.length ? (
                  room.history
                    .slice()
                    .reverse()
                    .map((h) => (
                      <details
                        className="round-detail"
                        key={h.number}
                        open={h.number === room.round}
                      >
                        <summary>
                          第 {h.number} 局 · {h.end}
                          <span>{h.scores.join(" / ")}</span>
                        </summary>
                        {h.wins?.map((w, i) => (
                          <p key={i}>
                            {room.players[w.seat]?.name} · {w.fan}番 ·{" "}
                            {w.reasons?.join("、")}
                          </p>
                        ))}
                        {h.ledger?.map((l, i) => (
                          <p key={i}>
                            {room.players[l.from]?.name} →{" "}
                            {room.players[l.to]?.name}：{l.amount}分（{l.reason}
                            ）
                          </p>
                        ))}
                      </details>
                    ))
                ) : (
                  <p className="muted">本局尚未完成。</p>
                )}
              </>
            )}
            {panel === "clubs" && user && (
              <TeaHouse
                user={user}
                defs={defs}
                clubs={clubs}
                api={api}
                work={work}
                refresh={async () => setClubs((await api("clubs")) || [])}
                create={async (clubId, templateId) => {
                  await command("create", { clubId, templateId });
                  setPanel("");
                }}
                join={async (id) => {
                  await command("join", { roomId: id });
                  setPanel("");
                }}
              />
            )}
            {panel === "report" && (
              <form
                onSubmit={(e) => {
                  e.preventDefault();
                  void work(async () => {
                    const v = await api("reports", {
                      roomId: room?.id || "",
                      text: form.report || "",
                    });
                    setPanel("");
                    flash("反馈已保存，编号 " + v.id);
                  });
                }}
              >
                <h2>意见反馈</h2>
                <p className="muted">玩法有出入，或遇到问题，请写在这里。</p>
                <textarea
                  placeholder="请描述问题与发生步骤（3–1000字）"
                  value={form.report || ""}
                  onChange={(e) => setForm({ ...form, report: e.target.value })}
                />
                <button className="primary full">保存反馈</button>
              </form>
            )}
            {panel === "admin" && (
              <>
                <h2>管理面板</h2>
                <form
                  onSubmit={(e) => {
                    e.preventDefault();
                    void work(async () => {
                      const r = await fetch("/api/admin", {
                        headers: { "X-Admin-Key": form.adminKey || "" },
                      });
                      const v = await r.json();
                      if (!r.ok) throw new Error(v.error);
                      setAdmin(v);
                    });
                  }}
                >
                  {field("adminKey", "管理密钥", "password")}
                  <button className="primary full">查看运行状态</button>
                </form>
                {admin && <AdminView data={admin} defs={defs} />}
              </>
            )}
            {panel === "tableMenu" && room && (
              <>
                <h2>房间 {room.id}</h2>
                <button
                  className="secondary full"
                  onClick={() => void openRules(room.config.game)}
                >
                  查看本桌游戏规则
                </button>
                <button
                  className="secondary full"
                  onClick={() => setPanel("settlement")}
                >
                  本桌战绩
                </button>
                <button
                  className="secondary full"
                  onClick={() => setPanel("report")}
                >
                  反馈问题
                </button>
                <button
                  className="danger full"
                  onClick={() =>
                    void work(async () => {
                      await command(
                        room.status === "waiting" ||
                          room.status === "finished" ||
                          room.status === "closed"
                          ? "leave"
                          : "dissolve",
                      );
                      setPanel("");
                      if (
                        room.status === "waiting" ||
                        room.status === "finished" ||
                        room.status === "closed"
                      )
                        setRoom(null);
                    })
                  }
                >
                  {room.status === "playing" || room.status === "roundEnd"
                    ? "申请解散房间"
                    : "离开房间"}
                </button>
                <p className="fine">
                  对局中解散需要全部真人同意。已完成局保留，未完成局的积分作废。
                </p>
              </>
            )}
          </section>
        </div>
      )}
    </main>
  );
}
createRoot(document.getElementById("root")!).render(<App />);

import React, { useEffect, useState } from "react";
type Config = {
  game: string;
  base: number;
  rounds: number;
  options: Record<string, number>;
};
type User = { id: string; name: string; avatar: number };
type Def = { key: string; name: string; players: number };
type Template = { id: string; name: string; enabled: boolean; config: Config };
type Club = {
  id: string;
  name: string;
  owner: string;
  members: Record<string, boolean>;
  managers?: Record<string, boolean>;
  disabled?: Record<string, boolean>;
  joinClosed: boolean;
  requireApproval?: boolean;
  autoCreate?: boolean;
  announcement?: string;
  templates?: Template[];
};
type Props = {
  user: User;
  defs: Def[];
  clubs: Club[];
  api: <T = any>(path: string, body?: any, method?: string) => Promise<T>;
  work: (fn: () => Promise<void>) => Promise<void>;
  refresh: () => Promise<void>;
  create: (clubId: string, templateId: string) => Promise<void>;
  join: (id: string) => Promise<void>;
};
const rounds = (game: string) =>
  game === "runfast"
    ? [10, 20]
    : game === "hongershi"
      ? [4, 8, 12]
      : game === "doushisi"
        ? [8, 12, 16]
        : [8, 16, 32];
const choices: Record<string, [string, string, number, [number, string][]][]> =
  {
    hongzhong: [
      [
        "root",
        "根加番",
        1,
        [
          [1, "开"],
          [0, "关"],
        ],
      ],
      [
        "ka",
        "卡加番",
        1,
        [
          [1, "开"],
          [0, "关"],
        ],
      ],
      [
        "minFan",
        "起胡",
        1,
        [
          [1, "1番"],
          [2, "2番"],
          [3, "3番"],
        ],
      ],
    ],
    sichuan: [
      [
        "root",
        "根加番",
        1,
        [
          [1, "开"],
          [0, "关"],
        ],
      ],
    ],
    hongershi: [
      [
        "kings",
        "王数",
        6,
        [
          [6, "6张"],
          [9, "9张"],
          [12, "12张"],
        ],
      ],
      [
        "redFive",
        "红五加红",
        0,
        [
          [1, "开"],
          [0, "关"],
        ],
      ],
      [
        "blackFive",
        "黑五加红",
        0,
        [
          [1, "开"],
          [0, "关"],
        ],
      ],
      [
        "selfEat",
        "吃牌范围",
        0,
        [
          [0, "双方摸翻"],
          [1, "仅自己摸翻"],
        ],
      ],
      [
        "cap",
        "封顶",
        4,
        [
          [4, "4番"],
          [5, "5番"],
          [6, "6番"],
        ],
      ],
      [
        "ladder",
        "阶梯",
        0,
        [
          [1, "开"],
          [0, "关"],
        ],
      ],
      [
        "checkListen",
        "查叫",
        0,
        [
          [1, "开"],
          [0, "关"],
        ],
      ],
      [
        "listenMode",
        "听牌范围",
        0,
        [
          [0, "含断张"],
          [1, "限剩余牌"],
        ],
      ],
    ],
    doushisi: [
      [
        "allowFloat",
        "漂分",
        1,
        [
          [1, "开"],
          [0, "关"],
        ],
      ],
      [
        "cap",
        "封顶分",
        500,
        [
          [500, "500"],
          [1000, "1000"],
        ],
      ],
      [
        "ladder",
        "阶梯",
        0,
        [
          [1, "开"],
          [0, "关"],
        ],
      ],
    ],
    runfast: [
      [
        "random",
        "减牌",
        0,
        [
          [0, "固定23张"],
          [1, "随机40张"],
        ],
      ],
      [
        "guan",
        "关牌翻倍",
        1,
        [
          [1, "开"],
          [0, "关"],
        ],
      ],
      [
        "bombCap",
        "炸弹封顶",
        1,
        [
          [-1, "不封顶"],
          [1, "1个"],
          [3, "3个"],
        ],
      ],
    ],
  };
export default function TeaHouse(p: Props) {
  const [selected, setSelected] = useState("");
  const [tab, setTab] = useState("大厅");
  const [detail, setDetail] = useState<any>(null);
  const [name, setName] = useState("");
  const [joinId, setJoinId] = useState("");
  const [notice, setNotice] = useState("");
  const [filter, setFilter] = useState("");
  const [memberFilter, setMemberFilter] = useState("");
  const [edit, setEdit] = useState<Template | null>(null);
  const [settings, setSettings] = useState({
    name: "",
    announcement: "",
    requireApproval: true,
    autoCreate: false,
    enabled: false,
  });
  const [confirm, setConfirm] = useState("");
  const club: Club | undefined = detail?.club;
  const manager =
    !!club && (club.owner === p.user.id || !!club.managers?.[p.user.id]);
  const owner = club?.owner === p.user.id;
  async function load(id = selected) {
    if (!id) return;
    const v = await p.api("club?id=" + encodeURIComponent(id));
    setDetail(v);
    setSettings({
      name: v.club.name,
      announcement: v.club.announcement || "",
      requireApproval: !!v.club.requireApproval,
      autoCreate: !!v.club.autoCreate,
      enabled: v.club.joinClosed,
    });
  }
  useEffect(() => {
    setTab("大厅");
    setDetail(null);
    setEdit(null);
    setConfirm("");
    setFilter("");
    setMemberFilter("");
    if (selected) void p.work(() => load(selected));
  }, [selected]);
  async function mutate(body: any) {
    await p.api("clubs", { id: selected, ...body });
    await p.refresh();
    if (body.op === "delete" || body.op === "leave") {
      setSelected("");
      setDetail(null);
    } else {
      await load();
    }
  }
  function act(body: any) {
    void p.work(() => mutate(body));
  }
  const defName = (key: string) =>
    p.defs.find((d) => d.key === key)?.name || key;
  const configText = (c: Config) =>
    `${defName(c.game)} · ${p.defs.find((d) => d.key === c.game)?.players}人 · ${c.rounds}局 · 底分${c.base}`;
  const options = (c: Config) =>
    (choices[c.game === "nohongzhong" ? "hongzhong" : c.game] || [])
      .map(
        ([key, label, fallback, values]) =>
          `${label}：${values.find((v) => v[0] === (c.options[key] ?? fallback))?.[1]}`,
      )
      .join(" / ");
  if (!selected)
    return (
      <section className="tea-home">
        <h2>茶馆</h2>
        <p className="muted">选择茶馆进入桌台大厅，馆内按已保存的玩法开桌。</p>
        <div className="tea-entry">
          <form
            onSubmit={(e) => {
              e.preventDefault();
              void p.work(async () => {
                const c = await p.api("clubs", {
                  op: "create",
                  name,
                  requireApproval: true,
                });
                await p.refresh();
                setSelected(c.id);
              });
            }}
          >
            <label>
              创建茶馆
              <input
                required
                maxLength={24}
                placeholder="茶馆名称"
                value={name}
                onChange={(e) => setName(e.target.value)}
              />
            </label>
            <button className="primary">创建茶馆</button>
          </form>
          <form
            onSubmit={(e) => {
              e.preventDefault();
              void p.work(async () => {
                const c = await p.api("clubs", {
                  op: "join",
                  id: joinId.trim(),
                });
                await p.refresh();
                if (c.pending)
                  setNotice(
                    "申请已提交，等待馆主或管理员审核。茶馆ID：" + c.id,
                  );
                else setSelected(c.id);
              });
            }}
          >
            <label>
              加入茶馆
              <input
                required
                placeholder="输入茶馆ID"
                value={joinId}
                onChange={(e) => setJoinId(e.target.value)}
              />
            </label>
            <button className="secondary">申请加入</button>
          </form>
        </div>
        {notice && <p role="status">{notice}</p>}
        <div className="tea-list">
          {p.clubs.map((c) => (
            <button
              key={c.id}
              className="tea-entry-card"
              onClick={() => setSelected(c.id)}
            >
              <strong>{c.name}</strong>
              <span>
                ID {c.id} · {Object.keys(c.members).length}名成员
              </span>
              <small>
                {c.owner === p.user.id
                  ? "我的茶馆"
                  : c.managers?.[p.user.id]
                    ? "管理员"
                    : "已加入"}
                {c.disabled?.[p.user.id] ? " · 已限制开桌/入桌" : ""}　进入 →
              </small>
            </button>
          ))}
        </div>
        {!p.clubs.length && <p className="muted">尚未加入茶馆。</p>}
      </section>
    );
  return (
    <section className="tea-house">
      <button
        className="text-link"
        onClick={() => {
          setSelected("");
          setDetail(null);
        }}
      >
        ‹ 我的茶馆
      </button>
      <div className="tea-heading">
        <div>
          <h2>{club?.name || "正在加载茶馆…"}</h2>
          <span className="muted">ID {selected} · 馆内娱乐积分</span>
        </div>
        <button className="secondary" onClick={() => void p.work(() => load())}>
          刷新
        </button>
      </div>
      {club && (
        <>
          <p className="tea-announcement">
            公告：{club.announcement || "欢迎入座，请文明游戏。"}
          </p>
          <nav className="tea-tabs">
            {[
              "大厅",
              "玩法",
              "成员",
              ...(manager ? ["审核"] : []),
              "战绩",
              ...(manager ? ["设置"] : []),
            ].map((t) => (
              <button
                className={tab === t ? "active" : ""}
                key={t}
                onClick={() => {
                  setTab(t);
                  setConfirm("");
                  setMemberFilter("");
                }}
              >
                {t}
                {t === "审核" ? ` (${detail.applications.length})` : ""}
              </button>
            ))}
          </nav>
          {tab === "大厅" && (
            <>
              <div className="tea-toolbar">
                <label>
                  玩法筛选
                  <select
                    value={filter}
                    onChange={(e) => setFilter(e.target.value)}
                  >
                    <option value="">全部玩法</option>
                    {club.templates?.map((t) => (
                      <option key={t.id} value={t.id}>
                        {t.name}
                      </option>
                    ))}
                  </select>
                </label>
                <span>
                  {
                    detail.rooms.filter(
                      (r: any) => !["finished", "closed"].includes(r.status),
                    ).length
                  }
                  张活动桌
                </span>
              </div>
              <div className="tea-tables">
                {(club.templates || [])
                  .filter((t) => t.enabled && (!filter || filter === t.id))
                  .map((t) => (
                    <article className="tea-table empty" key={t.id}>
                      <small>按馆内玩法新开桌</small>
                      <h3>{t.name}</h3>
                      <p>{configText(t.config)}</p>
                      <button
                        className="primary"
                        disabled={!!club.disabled?.[p.user.id]}
                        onClick={() =>
                          void p.work(() => p.create(club.id, t.id))
                        }
                      >
                        按此玩法开桌
                      </button>
                    </article>
                  ))}
                {detail.rooms
                  .filter(
                    (r: any) =>
                      !["finished", "closed"].includes(r.status) &&
                      (!filter || r.templateId === filter),
                  )
                  .map((r: any) => (
                    <article className="tea-table" key={r.id}>
                      <small>
                        房间 {r.id} ·{" "}
                        {r.status === "playing"
                          ? "对局中"
                          : r.status === "roundEnd"
                            ? "局间等待"
                            : "等待入座"}
                      </small>
                      <h3>
                        {club.templates?.find((t) => t.id === r.templateId)
                          ?.name || defName(r.config.game)}
                      </h3>
                      <p>{configText(r.config)}</p>
                      <strong>
                        {r.players} / {r.capacity} 人 · 第{r.round}局
                      </strong>
                      <button
                        className="secondary"
                        disabled={
                          (!!club.disabled?.[p.user.id] ||
                            r.players >= r.capacity ||
                            r.status !== "waiting") &&
                          !r.seated
                        }
                        onClick={() => void p.work(() => p.join(r.id))}
                      >
                        {r.seated
                          ? "返回牌桌"
                          : r.players >= r.capacity
                            ? "已满员"
                            : r.status !== "waiting"
                              ? "对局已开始"
                              : "加入房间"}
                      </button>
                      {manager && (
                        <>
                          <button
                            className="danger"
                            onClick={() =>
                              confirm === `room:${r.id}`
                                ? act({ op: "dissolveRoom", roomId: r.id })
                                : setConfirm(`room:${r.id}`)
                            }
                          >
                            {confirm === `room:${r.id}`
                              ? "确认解散房间"
                              : "管理员解散"}
                          </button>
                          {confirm === `room:${r.id}` && (
                            <small>
                              当前未结束的一局不结算，已结算战绩保留。
                              {club.autoCreate && "自动开房开启时会补建空桌。"}
                            </small>
                          )}
                        </>
                      )}
                    </article>
                  ))}
              </div>
              {!club.templates?.some((t) => t.enabled) && (
                <p className="muted">
                  茶馆还没有启用的玩法。
                  {manager ? "请在“玩法”中添加并启用。" : "请联系馆主设置。"}
                </p>
              )}
            </>
          )}
          {tab === "玩法" && (
            <>
              {manager && (
                <button
                  className="primary"
                  onClick={() =>
                    setEdit({
                      id: "",
                      name: "",
                      enabled: true,
                      config: {
                        game: "hongzhong",
                        base: 1,
                        rounds: 8,
                        options: {},
                      },
                    })
                  }
                >
                  添加玩法
                </button>
              )}
              {edit && (
                <form
                  className="tea-editor"
                  onSubmit={(e) => {
                    e.preventDefault();
                    void p.work(async () => {
                      await mutate({ op: "template", template: edit });
                      setEdit(null);
                    });
                  }}
                >
                  <h3>{edit.id ? "修改玩法" : "新增玩法"}</h3>
                  <label>
                    玩法名称
                    <input
                      required
                      maxLength={24}
                      value={edit.name}
                      onChange={(e) =>
                        setEdit({ ...edit, name: e.target.value })
                      }
                    />
                  </label>
                  <div className="tea-fields">
                    <label>
                      游戏
                      <select
                        value={edit.config.game}
                        onChange={(e) =>
                          setEdit({
                            ...edit,
                            config: {
                              game: e.target.value,
                              base: 1,
                              rounds: rounds(e.target.value)[0],
                              options: {},
                            },
                          })
                        }
                      >
                        {p.defs.map((d) => (
                          <option key={d.key} value={d.key}>
                            {d.name}（{d.players}人）
                          </option>
                        ))}
                      </select>
                    </label>
                    <label>
                      局数
                      <select
                        value={edit.config.rounds}
                        onChange={(e) =>
                          setEdit({
                            ...edit,
                            config: { ...edit.config, rounds: +e.target.value },
                          })
                        }
                      >
                        {rounds(edit.config.game).map((n) => (
                          <option key={n}>{n}</option>
                        ))}
                      </select>
                    </label>
                    <label>
                      底分
                      <select
                        value={edit.config.base}
                        onChange={(e) =>
                          setEdit({
                            ...edit,
                            config: { ...edit.config, base: +e.target.value },
                          })
                        }
                      >
                        {[1, 2, 5, 10].map((n) => (
                          <option key={n}>{n}</option>
                        ))}
                      </select>
                    </label>
                    {(
                      choices[
                        edit.config.game === "nohongzhong"
                          ? "hongzhong"
                          : edit.config.game
                      ] || []
                    ).map(([key, label, fallback, values]) => (
                      <label key={key}>
                        {label}
                        <select
                          value={edit.config.options[key] ?? fallback}
                          onChange={(e) =>
                            setEdit({
                              ...edit,
                              config: {
                                ...edit.config,
                                options: {
                                  ...edit.config.options,
                                  [key]: +e.target.value,
                                },
                              },
                            })
                          }
                        >
                          {values.map(([v, text]) => (
                            <option key={v} value={v}>
                              {text}
                            </option>
                          ))}
                        </select>
                      </label>
                    ))}
                  </div>
                  <label>
                    <input
                      type="checkbox"
                      checked={edit.enabled}
                      onChange={(e) =>
                        setEdit({ ...edit, enabled: e.target.checked })
                      }
                    />{" "}
                    启用此玩法
                  </label>
                  <div className="tea-buttons">
                    <button className="primary">保存玩法</button>
                    <button
                      type="button"
                      className="secondary"
                      onClick={() => setEdit(null)}
                    >
                      取消
                    </button>
                  </div>
                </form>
              )}
              {(club.templates || []).map((t) => (
                <article className="tea-row" key={t.id}>
                  <div>
                    <h3>
                      {t.name} · {t.enabled ? "已启用" : "已停用"}
                    </h3>
                    <p>{configText(t.config)}</p>
                    <small>{options(t.config)}</small>
                  </div>
                  {manager && (
                    <div className="tea-buttons">
                      <button
                        className="secondary"
                        onClick={() => setEdit(JSON.parse(JSON.stringify(t)))}
                      >
                        编辑
                      </button>
                      <button
                        className="secondary"
                        onClick={() =>
                          act({
                            op: "template",
                            template: { ...t, enabled: !t.enabled },
                          })
                        }
                      >
                        {t.enabled ? "停用" : "启用"}
                      </button>
                      <button
                        className="text-link"
                        onClick={() => setConfirm(t.id)}
                      >
                        删除
                      </button>
                      {confirm === t.id && (
                        <button
                          className="danger"
                          onClick={() => {
                            act({ op: "deleteTemplate", templateId: t.id });
                            setConfirm("");
                          }}
                        >
                          确认删除玩法
                        </button>
                      )}
                    </div>
                  )}
                </article>
              ))}
              <p className="muted">
                修改只影响之后新开的房间，已开桌继续使用开桌时的规则。
              </p>
            </>
          )}
          {tab === "成员" && (
            <>
              <input
                placeholder="搜索成员昵称或ID"
                value={memberFilter}
                onChange={(e) => setMemberFilter(e.target.value)}
              />
              {detail.members
                .filter((m: User) => (m.name + m.id).includes(memberFilter))
                .map((m: User) => (
                  <article className="tea-row" key={m.id}>
                    <div>
                      <strong>{m.name}</strong>
                      <small>
                        ID {m.id} ·{" "}
                        {m.id === club.owner
                          ? "馆主"
                          : club.managers?.[m.id]
                            ? "管理员"
                            : "成员"}
                        {club.disabled?.[m.id] ? " · 禁止开桌/入桌" : ""}
                      </small>
                      <small>
                        {detail.stats[m.id]?.rounds || 0}局 · 累计积分{" "}
                        {detail.stats[m.id]?.total || 0} · 大赢家{" "}
                        {detail.stats[m.id]?.bigWins || 0}次
                      </small>
                    </div>
                    <div className="tea-buttons">
                      {owner && m.id !== club.owner && (
                        <button
                          className="secondary"
                          onClick={() =>
                            act({
                              op: "manager",
                              userId: m.id,
                              enabled: !club.managers?.[m.id],
                            })
                          }
                        >
                          {club.managers?.[m.id] ? "取消管理" : "设为管理"}
                        </button>
                      )}
                      {manager &&
                        m.id !== club.owner &&
                        !club.managers?.[m.id] && (
                          <button
                            className="secondary"
                            onClick={() =>
                              act({
                                op: "disable",
                                userId: m.id,
                                enabled: !club.disabled?.[m.id],
                              })
                            }
                          >
                            {club.disabled?.[m.id] ? "恢复入桌" : "禁止入桌"}
                          </button>
                        )}
                      {manager &&
                        m.id !== club.owner &&
                        (owner || !club.managers?.[m.id]) && (
                          <>
                            <button
                              className="text-link"
                              onClick={() => setConfirm(m.id)}
                            >
                              移出
                            </button>
                            {confirm === m.id && (
                              <button
                                className="danger"
                                onClick={() => {
                                  act({ op: "remove", userId: m.id });
                                  setConfirm("");
                                }}
                              >
                                确认移出成员
                              </button>
                            )}
                          </>
                        )}
                    </div>
                  </article>
                ))}
              {!owner && (
                <button
                  className="danger"
                  onClick={() =>
                    confirm === "leave"
                      ? act({ op: "leave" })
                      : setConfirm("leave")
                  }
                >
                  {confirm === "leave" ? "确认退出茶馆" : "退出茶馆"}
                </button>
              )}
            </>
          )}
          {tab === "审核" && manager && (
            <>
              {!detail.applications.length && (
                <p className="muted">没有待审核申请。</p>
              )}
              {detail.applications.map((a: any) => (
                <article className="tea-row" key={a.user.id}>
                  <div>
                    <strong>{a.user.name}</strong>
                    <small>
                      ID {a.user.id} · {new Date(a.at * 1000).toLocaleString()}
                    </small>
                  </div>
                  <div className="tea-buttons">
                    <button
                      className="primary"
                      onClick={() => act({ op: "approve", userId: a.user.id })}
                    >
                      同意
                    </button>
                    <button
                      className="secondary"
                      onClick={() => act({ op: "reject", userId: a.user.id })}
                    >
                      拒绝
                    </button>
                  </div>
                </article>
              ))}
            </>
          )}
          {tab === "战绩" && (
            <>
              <label>
                查成员战绩
                <select
                  value={memberFilter}
                  onChange={(e) => setMemberFilter(e.target.value)}
                >
                  <option value="">全部成员</option>
                  {detail.members.map((m: User) => (
                    <option key={m.id} value={m.id}>
                      {m.name} · {m.id}
                    </option>
                  ))}
                </select>
              </label>
              {detail.history
                .filter(
                  (h: any) =>
                    !memberFilter ||
                    h.players.some((m: any) => m?.userId === memberFilter),
                )
                .map((h: any) => (
                  <details className="round-detail" key={h.room.id}>
                    <summary>
                      房间 {h.room.id} · {defName(h.room.config.game)} ·{" "}
                      {h.rounds.length}局 ·{" "}
                      {h.room.status === "finished"
                        ? "已完成"
                        : h.room.status === "closed"
                          ? "已解散"
                          : "进行中"}
                    </summary>
                    <p>
                      {h.players.map((m: any) => m?.name || "空位").join(" / ")}
                    </p>
                    {h.rounds.map((r: any) => (
                      <details key={r.number}>
                        <summary>
                          第{r.number}局 · {r.end} · {r.scores.join(" / ")}
                        </summary>
                        {r.wins?.map((w: any, i: number) => (
                          <p key={i}>
                            {h.players[w.seat]?.name}：{w.fan}番 ·{" "}
                            {w.reasons?.join("、")}
                          </p>
                        ))}
                        {r.ledger?.map((l: any, i: number) => (
                          <p key={i}>
                            {h.players[l.from]?.name} → {h.players[l.to]?.name}
                            ：{l.amount}分（{l.reason}）
                          </p>
                        ))}
                      </details>
                    ))}
                  </details>
                ))}
              {!detail.history.length && (
                <p className="muted">尚无已结算的小局。</p>
              )}
            </>
          )}
          {tab === "设置" && manager && (
            <>
              <form
                className="tea-editor"
                onSubmit={(e) => {
                  e.preventDefault();
                  act({ op: "settings", ...settings });
                }}
              >
                <label>
                  茶馆名称
                  <input
                    required
                    maxLength={24}
                    value={settings.name}
                    onChange={(e) =>
                      setSettings({ ...settings, name: e.target.value })
                    }
                  />
                </label>
                <label>
                  公告
                  <textarea
                    maxLength={300}
                    value={settings.announcement}
                    onChange={(e) =>
                      setSettings({ ...settings, announcement: e.target.value })
                    }
                  />
                </label>
                <label>
                  <input
                    type="checkbox"
                    checked={settings.requireApproval}
                    onChange={(e) =>
                      setSettings({
                        ...settings,
                        requireApproval: e.target.checked,
                      })
                    }
                  />{" "}
                  加入需管理员审核
                </label>
                <label>
                  <input
                    type="checkbox"
                    checked={settings.enabled}
                    onChange={(e) =>
                      setSettings({ ...settings, enabled: e.target.checked })
                    }
                  />{" "}
                  暂停接受新成员
                </label>
                <label>
                  <input
                    type="checkbox"
                    checked={settings.autoCreate}
                    onChange={(e) =>
                      setSettings({ ...settings, autoCreate: e.target.checked })
                    }
                  />{" "}
                  无空桌时自动开房
                </label>
                <p className="muted">
                  每个启用玩法保留一张无人空桌，首位入座者为房主。关闭后已建房间保留；刷新大厅查看空桌。
                </p>
                <button className="primary">保存设置</button>
              </form>
              {owner && (
                <div className="tea-editor">
                  <p>解散前请关闭自动开房，并结束馆内所有房间。</p>
                  <button
                    className="danger"
                    onClick={() =>
                      confirm === "delete"
                        ? act({ op: "delete" })
                        : setConfirm("delete")
                    }
                  >
                    {confirm === "delete" ? "确认解散茶馆" : "解散茶馆"}
                  </button>
                </div>
              )}
            </>
          )}
        </>
      )}
    </section>
  );
}

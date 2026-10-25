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
  externalRecordsEnabled?: boolean;
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
  const [roomFilter, setRoomFilter] = useState("all");
  const [expandedRoom, setExpandedRoom] = useState("");
  const [memberFilter, setMemberFilter] = useState("");
  const [externalToken, setExternalToken] = useState("");
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
    setRoomFilter("all");
    setExpandedRoom("");
    setMemberFilter("");
    setExternalToken("");
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
  function manageExternalRecords(op: "issueRecordsToken" | "revokeRecordsToken") {
    void p.work(async () => {
      if (op === "issueRecordsToken") {
        const result = await p.api<{ token: string }>("clubs", { id: selected, op });
        setExternalToken(result.token);
      } else {
        await p.api("clubs", { id: selected, op });
        setExternalToken("");
      }
      await p.refresh();
      await load();
    });
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
  const activeRooms = (detail?.rooms || []).filter(
    (r: any) => !["finished", "closed"].includes(r.status),
  );
  const visibleRooms = activeRooms.filter((r: any) => {
    if (filter && r.templateId !== filter) return false;
    if (roomFilter === "playing")
      return r.status === "playing" || r.status === "roundEnd";
    if (roomFilter === "joinable")
      return (
        r.status === "waiting" &&
        r.players < r.capacity &&
        !r.seated &&
        !club?.disabled?.[p.user.id]
      );
    return true;
  });
  if (!selected)
    return (
      <section className="tea-home">
        <h2>茶馆</h2>
        <p className="muted">管理亲友茶馆、加入熟人牌局；进入后可按馆内玩法开桌。</p>
        {notice && <p className="tea-notice" role="status">{notice}</p>}
        <div className="tea-overview">
          <section className="tea-overview-panel my-tea-panel">
            <h3><span>我的茶馆</span><small>{p.clubs.length} 个</small></h3>
            <div className="tea-overview-content">
              <div className="tea-list">
                {p.clubs.map((c) => (
                  <button
                    key={c.id}
                    className="tea-entry-card"
                    onClick={() => setSelected(c.id)}
                  >
                    <strong>{c.name}</strong>
                    <span>茶馆ID　{c.id}</span>
                    <small>
                      {Object.keys(c.members).length} 名成员 · {c.owner === p.user.id
                        ? "馆主"
                        : c.managers?.[p.user.id]
                          ? "管理员"
                          : "成员"}
                      {c.disabled?.[p.user.id] ? " · 已限制开桌/入桌" : ""}　进入 →
                    </small>
                  </button>
                ))}
                {!p.clubs.length && (
                  <p className="tea-empty-state"><span>♧</span>暂未拥有或加入茶馆。</p>
                )}
              </div>
              <form
                className="tea-create-inline"
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
                    placeholder="输入茶馆名称"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                  />
                </label>
                <button className="primary">创建茶馆</button>
              </form>
            </div>
            <footer>已加入：{p.clubs.length} 个</footer>
          </section>
          <section className="tea-overview-panel join-tea-panel">
            <h3><span>加入茶馆</span><small>输入茶馆ID</small></h3>
            <div className="tea-overview-content join-tea-content">
              <div className="tea-join-prompt">
                <span aria-hidden="true">⌂</span>
                <strong>和熟悉的牌友一起开桌</strong>
                <small>提交后按茶馆设置加入或等待审核</small>
              </div>
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
                      setNotice("申请已提交，等待馆主或管理员审核。茶馆ID：" + c.id);
                    else setSelected(c.id);
                  });
                }}
              >
                <label>
                  茶馆ID
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
            <footer>加入后可查看馆内玩法和开放牌桌</footer>
          </section>
        </div>
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
            <div className="tea-detail-layout">
              <aside className="tea-info-panel">
                <div className="tea-info-plaque">茶馆信息</div>
                <dl>
                  <div><dt>馆名</dt><dd>{club.name}</dd></div>
                  <div><dt>ID</dt><dd>{club.id}</dd></div>
                  <div><dt>馆主ID</dt><dd>{club.owner}</dd></div>
                  <div><dt>成员</dt><dd>{Object.keys(club.members || {}).length} 人</dd></div>
                  <div><dt>我的身份</dt><dd>{owner ? "馆主" : manager ? "管理员" : "成员"}</dd></div>
                  <div><dt>加入方式</dt><dd>{club.requireApproval ? "需审核" : "直接加入"}</dd></div>
                </dl>
                <div className="tea-info-actions">
                  <button onClick={() => setTab("战绩")}>查看战绩　→</button>
                  <button onClick={() => setTab("玩法")}>馆内玩法　→</button>
                </div>
              </aside>
              <section className="tea-room-panel">
                <div className="tea-room-heading">
                  <div><h3>茶馆牌桌</h3><small>浏览活动桌，或从馆内玩法创建新桌</small></div>
                  <strong>{activeRooms.length} 张活动桌</strong>
                </div>
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
                <div className="tea-room-filters" role="group" aria-label="牌桌筛选">
                  {[["all", "全部"], ["playing", "游戏中"], ["joinable", "可加入"]].map(([key, label]) => (
                    <button
                      key={key}
                      className={roomFilter === key ? "active" : ""}
                      onClick={() => setRoomFilter(key)}
                    >{label}</button>
                  ))}
                </div>
              </div>
              <div className="tea-tables">
                {(roomFilter === "all" ? club.templates || [] : [])
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
                {visibleRooms
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
                        className="tea-room-details-toggle"
                        aria-expanded={expandedRoom === r.id}
                        onClick={() =>
                          setExpandedRoom((current) =>
                            current === r.id ? "" : r.id,
                          )
                        }
                      >
                        牌桌详情 {expandedRoom === r.id ? "⌃" : "⌄"}
                      </button>
                      {expandedRoom === r.id && (
                        <dl className="tea-room-details">
                          <div>
                            <dt>房间号</dt>
                            <dd>{r.id}</dd>
                          </div>
                          <div>
                            <dt>牌桌状态</dt>
                            <dd>
                              {r.status === "playing"
                                ? "对局中"
                                : r.status === "roundEnd"
                                  ? "局间等待"
                                  : "等待入座"}
                            </dd>
                          </div>
                          <div>
                            <dt>当前进度</dt>
                            <dd>
                              {r.players}/{r.capacity} 人 · 第{r.round}局
                            </dd>
                          </div>
                          <div>
                            <dt>房间玩法</dt>
                            <dd>{configText(r.config)}</dd>
                          </div>
                        </dl>
                      )}
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
                {!visibleRooms.length && (roomFilter !== "all" || !club.templates?.some((t) => t.enabled)) && (
                  <p className="tea-empty-rooms">{roomFilter === "playing" ? "当前没有进行中的牌桌。" : roomFilter === "joinable" ? "当前没有可加入的牌桌。" : "暂无活动牌桌。"}</p>
                )}
              </div>
              {!club.templates?.some((t) => t.enabled) && (
                <p className="muted">
                  茶馆还没有启用的玩法。
                  {manager ? "请在“玩法”中添加并启用。" : "请联系馆主设置。"}
                </p>
              )}
              </section>
            </div>
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
              <p className="muted">显示本茶馆最近7天已结算的小局；过期战绩会自动清除。</p>
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
              <div className="tea-editor">
                <h3>外部战绩接口</h3>
                <p>
                  {club?.externalRecordsEnabled
                    ? "只读接口已启用。生成新密钥会立即替换旧密钥。"
                    : "生成只读密钥后，外部服务可读取本茶馆最近7天战绩。"}
                </p>
                <p className="muted">
                  接口地址：{window.location.origin}/api/v1/teahouses/{selected}/records
                </p>
                <button
                  className="primary"
                  onClick={() => manageExternalRecords("issueRecordsToken")}
                >
                  {club?.externalRecordsEnabled ? "轮换接口密钥" : "生成接口密钥"}
                </button>{" "}
                {club?.externalRecordsEnabled && (
                  <button
                    className="danger"
                    onClick={() => manageExternalRecords("revokeRecordsToken")}
                  >
                    撤销接口密钥
                  </button>
                )}
                {externalToken && (
                  <label>
                    新密钥（仅显示本次）
                    <textarea readOnly value={externalToken} rows={2} />
                  </label>
                )}
                <small className="muted">
                  外部服务通过 Authorization: Bearer &lt;密钥&gt; 调用。请只经 HTTPS 传输并妥善保存密钥；微信小程序由接入方开发，需自行配置合法请求域名。
                </small>
              </div>
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

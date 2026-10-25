> 历史来源分析（2026-10-23）：保留原包和截图证据，文中缺口及阶段限制不代表当前实现。当前六款规则以 docs/rules/CLIENT_REVIEW_RULES_SIX_GAMES_2026-10-24.txt 为准，交付状态见 docs/PROGRESS.md。

# 页面清单

证据口径：C=可读客户端代码；R=配置/资源；B=实际运行原客户端；I=推断；U=缺失。未运行原客户端，所有 B 均为未验证。静态反编译可能有误，不能等同服务器源码。证据根目录为 `analysis/extracted/majiang-readable-handoff/work/decoded/`。下文 G=game/game，Z=plaza/plaza/src。原包仅内部参考，不进入产品。

|页面|原包证据（相对Z）|结构与功能|存在程度|H5映射|
|---|---|---|---|---|
|启动/热更新|ClientScene.lua、shell加载器|动态游戏下载/版本|部分存在|本地PWA启动|
|登录/注册/重置|views/layer/logon/Logon*.lua、Register*.lua|账号/手机/站点选择|客户端完整，服务端缺失|用户名/密码登录注册；无重置|
|大厅/游戏菜单|views/layer/plaza/GameListLayer.lua|右侧滚动游戏，左侧加房/创房、底栏、公告|客户端完整|五游戏入口|
|头像/个人资料|views/layer/modifyuserinfo/ModifyInfo*.lua|昵称/头像/密码|客户端完整|昵称/原创头像|
|创建房间|views/layer/battle/NBattleCreateLayer.lua、CreateConfigView.lua|SubGameListView、控件组、在线optionInfo|部分存在|证据配置/待验证提示|
|加入/邀请|BattleFindLayer、GameClientEngine邀请函数|房号输入/分享文本|客户端存在，分享SDK依赖|房号/链接复制|
|茶馆大厅/桌子|views/layer/teahouse/TH_Main.lua、TeaHouse.lua、TableView.lua|桌子、成员、规则、列表|客户端完整，服务端缺失|茶馆房间列表|
|茶馆创建/加入|CreateDlg.lua、JoinDlg.lua|名称/编号|客户端完整|真实持久化|
|茶馆成员/管理|MemberDlg、ManagerDlg、ManagerSet、ApplyListDlg、OwnerDlg|申请/角色/成员/设置|客户端存在|加入、角色、移除、设置|
|茶馆统计/战绩|BattleDetail、UserBattle、CalculateDlg、BigWinner、ActualTimeInfo|按用户/桌统计|客户端存在，算法服务端|真实本地战绩|
|游戏桌面|各游戏 views/GameViewLayer.lua、HandCard*.lua|座位/手牌/操作/出牌/听牌|四个目标模块存在|按证据牌类独立UI|
|红二十桌面|仅dfex ID110入口|动态下载项|只有配置|缺失说明，禁止伪造|
|结算/回放|GameEndLayer、GameEndRecord、GameVideo模块|单局/总分/录像|客户端存在，结算服务端|胜者/余牌实录，积分未验证|
|战绩|BattleList/Score/Record、茶馆UserBattle|查询/分页/分享|客户端存在|服务器战绩|
|设置/音效|SettingLayer及各GameView设置|音乐/音效/退出|客户端存在|本地偏好|
|公告/客服/举报|RichNotify、help/service入口|公告/客服|公告有引用，独立举报未确认|公告；举报为新系统技术验证扩展|
|商店/银行/兑换/福利|GameListLayer底栏与wealth模块|货币/商城|客户端存在|不接入现金或购买|

全部181个UI文件的图片、音效引用与构造/点击函数见 analysis/UI_RESOURCE_REFERENCES.json；5509个函数的位置见 FUNCTION_INDEX.json。未恢复的布局数值和图像不声称已视觉验证。

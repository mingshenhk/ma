"""Write the analysis checkpoint from independently recorded observations."""
from pathlib import Path
import json, re
P=Path(__file__).resolve().parents[2]
D=P/'docs'
(D/'rules').mkdir(parents=True,exist_ok=True)
def save(name,text):
    f=P/name;f.parent.mkdir(parents=True,exist_ok=True);f.write_text(text.strip()+'\n',encoding='utf-8')
intro='''证据口径：C=可读客户端代码；R=配置/资源；B=实际运行原客户端；I=推断；U=缺失。未运行原客户端，所有 B 均为未验证。静态反编译可能有误，不能等同服务器源码。证据根目录为 `analysis/extracted/majiang-readable-handoff/work/decoded/`。下文 G=game/game，Z=plaza/plaza/src。原包仅内部参考，不进入产品。'''
fields=['游戏名称及内部 ID','玩家人数','使用的牌','牌组数量','初始牌数','发牌方式','座位及轮次','庄家规则','可执行动作','特殊牌型','合法性判断','游戏结束条件','计分和结算','房间可选规则','局数设置','特殊玩法配置','原包实现证据','尚未确定的问题']
games=[
('01_hongzhong_duanqiaqou','红中断掐勾麻将',[
'R：候选对应 ID 150、sparrowhzdgk，包内名“红中断勾卡”。客户“掐”与原包“卡”是否同义待确认。',
'C：PLAYER_COUNT=4，GAME_PLAYER=4。',
'C：两门数字牌 0x11..0x19、0x21..0x29；红中 0x35 为癞子。不是三门四川麻将。',
'C/I：CARD_COUNT=72+4；18种数字各4张+4红中=76。MAX_REPERTORY=72 与此冲突，需服务器验证。',
'C：MAX_COUNT=11；胡牌分析接收2、5、8、11张，三组面子加一对的结构。开局庄闲数量仅客户端容量/消息可见，未有服务器发牌实现。',
'U：服务器给牌；洗牌、补牌、牌墙顺序与分包方式缺失。',
'C：四座，当前用户/提供者由服务器消息指定；完整动作优先级 U。',
'U：服务器发送庄家，连庄、换庄及首庄选取不在包内。',
'C：出牌、碰、杠、加杠、报胡、胡/点炮及托管消息；红中不可碰杠由 GameView 文本确认。吃牌枚举存在不代表房间实际允许。',
'C：平胡、对对、清一色、五对/龙五对、将/清组合、带幺、断幺、天/地、杠开、杠炮、抢杠、海底、报叫等标志；是否全部启用依赖服务器。',
'C：GameLogic AnalyseCard、AnalyseBao、SimpleAnalyseChiHu 等；红中作为替牌。不能据此确认所有过胡/抢杠边界。',
'C：HU_RESULT、GAME_END 由服务器；U：血战续局、牌墙耗尽及离线终局细则。',
'U：结果包含服务器分数；客户端番提示不是完整付款表。掐/根是否加番存在注释和禁用代码，不能激活。',
'R/C：NBattleCreateLayer 从服务器取得 optionInfo/specialInfo；不含完整实际选项表。',
'U：通用房间字段支持局数，具体允许值从在线配置获取，包内未固定。',
'C：cbMagicCard=0x35、cbMagicIndex=28；报胡与清胡等操作位存在，互斥和优先级 U。',
'Z/models/dfex.lua 游戏表 ID150；G/chess/sparrowhzdgk/src/models/CMD_Game.lua；同模块 GameLogic.lua、GameClientEngine.lua、views/GameViewLayer.lua（约527行说明红中）。GameLogic AnalyseBao约410、AnalyseCard约485、SimpleAnalyseChiHu约993。',
'名称映射、76/72矛盾、完整发牌、庄家、动作优先级、断掐勾语义、完整结算、选项表和实机行为。'
]),
('02_no_hongzhong_duanqiaqou','不带红中断掐勾麻将',[
'R：候选 ID57 sparrowsclm“断勾卡血战麻将”；ID114 sparrowerdgk 是二人断勾卡，不能自动代替客户玩法。',
'C：sparrowsclm PLAYER_COUNT=3；GAME_PLAYER=4 仅协议容量。114为2人。',
'C：0x11..0x19、0x21..0x29，不含0x35红中。',
'C/I：18种×4=72，MAX_REPERTORY=72。',
'C：MAX_COUNT=11、MAX_WEAVE=3，胡牌为三组加一对。庄闲实发数量 U。',
'U：服务器负责洗牌与发牌；本地 RandCardList 不能证明线上发牌策略。',
'C：三人逻辑和四槽协议分离；实际换手由服务器。',
'U：首庄、连庄、胡后换庄均缺失。',
'C：出牌、碰、杠、加杠、报胡、胡、托管；吃牌仅枚举不足以确认开放。',
'C：平胡、对对、清一色、五对/龙五对、将对、带幺等；不能套用13张七对。',
'C：AnalyseChiHuCard约678、AnalyseWuDui约777、AnalyseQing约823；GetQiaCount约111与 GetGengCount约139。',
'U：服务器结束消息和续局消息存在，但完整结束条件缺失。',
'C：GetHuRank约263..400提示平胡1、对对2、清一色/五对3、清对/龙五对4、带幺/清五对5，若干组合6；抢杠/杠开/杠炮/海底/断幺各加1、报叫加2。U：付款方、倍率、封顶和最终支付。根/掐加番调用被注释。',
'U：每种房间实际 optionInfo 从服务端下发；通用配置不能作为该玩法选项。',
'U：具体可选局数缺失。',
'C：掐的客户端函数围绕5点中张；非自摸且当前牌非5返回0；根统计手牌/面子四张。此为函数语义，不代表当前结算启用。',
'Z/models/dfex.lua ID57、114；G/chess/sparrowsclm/src/models/CMD_Game.lua、GameLogic.lua 及 GameClientEngine.lua。',
'客户是否选择3人57还是2人114、名称映射、完整结算/庄家/操作顺序及选项。'
]),
('03_hongershi','红二十',[
'R：ID110、platehes、recreational、kindname=poker，仅见大厅动态下载表。',
'U：未发现。','U：未发现，不能因 poker 标签推断标准扑克牌。','U：未发现。','U：未发现。','U：未发现。','U：未发现。','U：未发现。','U：未发现。','U：未发现。','U：未发现。','U：未发现。','U：未发现。','U：未发现。','U：未发现。','U：未发现。',
'Z/models/dfex.lua ID110。全量8253文件名、366份可读Lua及4个嵌套ZIP目录未发现platehes模块；NESTED_ARCHIVES.json/SEMANTIC_SCAN.json有机器记录。chexuan属于其他游戏，不是红二十。',
'全部关键规则与原客户端资源缺失。精确名称外部检索没有找到可验证的原厂规则；不采用同名网页或其他地区玩法。需要同版本客户端补丁、规则书或原服测试录像。'
]),
('04_paodekuai','跑得快',[
'C/R：ID76、runfast；ID115 errunfast为另一二人版本。',
'C：默认4人，存在3人配置分支；本轮可验证实现仅开放4人。',
'C：四花色 A、5、6、7、8、9、10、J、Q、K；无2/3/4/大小王。A逻辑值14。',
'C：FULL_COUNT=40，HAND_COUNT=10，40个独立牌码。',
'C：四人每人10张。3人减牌策略有随机/固定配置但完整规则 U。',
'U：实际洗牌由服务器；四人全部发完的容量和开局数量可确认。',
'C：按座位轮次，首轮含黑桃5；所有玩家手牌10时设置首出标记。',
'C：首出必须包含0x35；下一局先手规则 U。',
'C：出牌、过牌、托管；不能领出时空过；强制过还是可主动过的服务端规则 U。',
'C：单、对、三张、顺子≥3、连对≥2组、连续三张≥2组、四张炸弹。不含三带/四带/飞机带翅膀。CLOSE枚举不能证实可主动使用。',
'C：CompareCard约451..511，同类型同长度比最高逻辑值，炸弹压非炸弹。下家1张时单张必须最大；第一出含黑桃5。客户端早退/死代码不照搬。',
'C/I：手牌清空后服务器发结束；本地实现可检验出完，但原服是否另有“关门”结束 U。',
'U：GAME_END 读取余牌、炸弹、分数；没有服务器计分公式。本轮仅记胜者/余牌和炸弹次数，不产生自创积分。',
'C：3/4人、炸弹封顶/255不封顶、3人随机/固定减牌；封顶计分缺失不开放。',
'U：允许局数由在线配置；本轮1/4/8为工程验证设置，明确非还原结论。',
'C：黑桃5首出、报单顶大。U：强制压牌、封顶结算、首局之后领出。',
'G/recreational/runfast/src/models/CMD_Game.lua；assist/GameLogic.lua GetCardType约103、CompareCard约451；GameClientEngine.lua onSubGameStart约764、onSubGameEnd约873、VerdictOutCard约2020..2085、邀请文本约1750。',
'三人减牌、计分、强制过牌、关门规则、跨局先手。验证配置以“待验证”展示，不能称完整原服复刻。'
]),
('05_doushisi','斗十四',[
'R：ID95、platedss、recreational；是中国骨牌，不是标准扑克牌。',
'C：默认4人，界面有2/4人，server wPlayerCount决定；并非网上同名3人牌例。',
'C：21种牌码1111、1102、0202、1103、0203、1114、0303、1105、0214、0205、1106、0314、0206、0305、1414、0306、1405、0505、1406、0506、1606。',
'U：21是种类数；遍历听牌使用每种<4不足以证明完整牌组张数。不得自行补成84/88/56张。',
'C/I：恢复公式11-2×明组，当前用户另+1；推测闲11、庄12，但没有发牌端证明。',
'U：洗牌、底牌、补牌和偷牌时机缺失。',
'C：current/source字段和座位UI存在；优先级与跳转规则 U。',
'U：服务器给庄家；选庄/换庄未发现。',
'C：吃（两牌点数和14）、明拉、暗拉、报、胡；steal通知。',
'C：平/地/天/黑/红胡标志；AnalyseChiHuCard黑/红附加检查实际被注释，不能重新激活。',
'C：AnalyseCard 将余手全部拆成点数和14的两张组；GetAnCard识别≥3同牌。',
'U：GAME_END 由服务器，完整终局条件未发现。',
'C：红点表、明/暗拉最低10点、默认门槛BASE_POINT=30可配置；U：完整胡分、番、支付公式。',
'C：BASE_POINT可设、人数2/4；具体合法范围和其余服务器选项 U。',
'U：允许局数从服务器配置。',
'C：GetCardsLogicPoint、GetUserRedPoint、AnalyseHearCard/ReportCard门槛；偷牌仅消息接收。',
'G/recreational/platedss/src/assist/GameLogic.lua 常量表、GetEatCards、GetAnCard、803..1095；GameClientEngine.lua 160..193、274..303、377..442、1021/1076及恢复1236；views/GameViewLayer.lua约1412/1420。',
'牌组重复数、实发、庄家、偷牌、优先级、完整结算。只发布确认的组合分析器，联机开局关闭并说明缺口。'
])]
for slug,title,values in games:
    assert len(values)==18
    save('docs/rules/'+slug+'.md','# '+title+'\n\n'+intro+'\n\n'+'\n\n'.join('## '+str(i+1)+'. '+k+'\n\n'+v for i,(k,v) in enumerate(zip(fields,values))))
save('analysis/TECHNOLOGY_REPORT.md','''# 技术识别与环境报告

- 原ZIP 281,750,936字节；SHA-256 `7f1c08d8f1f1843adadfe890c39d1c9052a142064b7d0087fc995f43db63a598`；8253个文件；449,311,036字节解压量；CRC全部通过；路径穿越/加密条目0。完整机器结果见 ARCHIVE_REPORT.json。
- 内容是已有反编译交接包，不是可构建的原工程。Android包名 com.foxuc.scjyproject，Cocos2d-x + Lua，Java启动和SDK壳、JNI原生库；不是Cocos Creator或Unity工程。
- 894个.lua扩展名含压缩/原始字节码；366份可读Lua（大厅167、shell79、棋牌游戏69、休闲51）。DEX2、SO4、Java2144、Smali2216；原生C++源/服务器源未发现。字节码反编译的控制流需核对，不能视为源码正确性证明。
- 1981个.png中仅57有标准PNG头。大部分资源有自定义压缩/加密；19个plist，另有MP3 218、WAV535。图集引用可定位，原始大场景无法直接视觉复现。原包解码工具采用原生SBOX/Huffman机制，但未执行未知工具或APK；不绕过在线鉴权。
- 4个嵌套ZIP为game/plaza两份重复，CRC均通过；清单再次扫描，无红二十模块。不存在CSB/CSD原始布局工程；主要UI由Lua程序化定位。
- 技术证据：work/decoded/plaza/plaza/src、game/game各模块；work/extracted/pkg/AndroidManifest.xml；JADX/Smali/so反编译资料；完整路径和头部摘要见FILE_INVENTORY.csv。
- 工具：Windows/PowerShell，Python3.11.4、Node24.15、npm、git2.54、rg、Java；Docker CLI29.7.2存在，daemon未启动；Go未在PATH。只安装构建/测试需要的官方工具。
- 不运行未知包程序，不连接原服，不导出原包配置中的凭据；产品重新定义协议与品牌资源。
''')
screens=[
('启动/热更新','ClientScene.lua、shell加载器','动态游戏下载/版本','部分存在','本地PWA启动'),
('登录/注册/重置','views/layer/logon/Logon*.lua、Register*.lua','账号/手机/站点选择','客户端完整，服务端缺失','用户名/密码登录注册；无重置'),
('大厅/游戏菜单','views/layer/plaza/GameListLayer.lua','右侧滚动游戏，左侧加房/创房、底栏、公告','客户端完整','五游戏入口'),
('头像/个人资料','views/layer/modifyuserinfo/ModifyInfo*.lua','昵称/头像/密码','客户端完整','昵称/原创头像'),
('创建房间','views/layer/battle/NBattleCreateLayer.lua、CreateConfigView.lua','SubGameListView、控件组、在线optionInfo','部分存在','证据配置/待验证提示'),
('加入/邀请','BattleFindLayer、GameClientEngine邀请函数','房号输入/分享文本','客户端存在，分享SDK依赖','房号/链接复制'),
('茶馆大厅/桌子','views/layer/teahouse/TH_Main.lua、TeaHouse.lua、TableView.lua','桌子、成员、规则、列表','客户端完整，服务端缺失','茶馆房间列表'),
('茶馆创建/加入','CreateDlg.lua、JoinDlg.lua','名称/编号','客户端完整','真实持久化'),
('茶馆成员/管理','MemberDlg、ManagerDlg、ManagerSet、ApplyListDlg、OwnerDlg','申请/角色/成员/设置','客户端存在','加入、角色、移除、设置'),
('茶馆统计/战绩','BattleDetail、UserBattle、CalculateDlg、BigWinner、ActualTimeInfo','按用户/桌统计','客户端存在，算法服务端','真实本地战绩'),
('游戏桌面','各游戏 views/GameViewLayer.lua、HandCard*.lua','座位/手牌/操作/出牌/听牌','四个目标模块存在','按证据牌类独立UI'),
('红二十桌面','仅dfex ID110入口','动态下载项','只有配置','缺失说明，禁止伪造'),
('结算/回放','GameEndLayer、GameEndRecord、GameVideo模块','单局/总分/录像','客户端存在，结算服务端','胜者/余牌实录，积分未验证'),
('战绩','BattleList/Score/Record、茶馆UserBattle','查询/分页/分享','客户端存在','服务器战绩'),
('设置/音效','SettingLayer及各GameView设置','音乐/音效/退出','客户端存在','本地偏好'),
('公告/客服/举报','RichNotify、help/service入口','公告/客服','公告有引用，独立举报未确认','公告；举报为新系统技术验证扩展'),
('商店/银行/兑换/福利','GameListLayer底栏与wealth模块','货币/商城','客户端存在','不接入现金或购买')]
head='|页面|原包证据（相对Z）|结构与功能|存在程度|H5映射|\n|---|---|---|---|---|\n'
save('docs/UI_SCREEN_INVENTORY.md','# 页面清单\n\n'+intro+'\n\n'+head+'\n'.join('|'+ '|'.join(x)+'|' for x in screens)+'\n\n全部181个UI文件的图片、音效引用与构造/点击函数见 analysis/UI_RESOURCE_REFERENCES.json；5509个函数的位置见 FUNCTION_INDEX.json。未恢复的布局数值和图像不声称已视觉验证。')
save('docs/FEATURE_MATRIX.md','# 功能矩阵\n\n'+head+'\n'.join('|'+ '|'.join(x)+'|' for x in screens)+'\n\n“客户端完整”仅指页面和调用链存在，服务器实现均未随包提供。源界面并非可直接复用的H5。')
save('docs/NAVIGATION_MAP.md','''# 导航关系

原包：启动/热更新 → Logon → ClientScene → GameListLayer。大厅 → BattleFind（加房）/NBattleCreate（创房）→ GameClientEngine → GameView → GameEnd → 房间/大厅。大厅 → TeaHouse → TH_Main → 桌/成员/管理/战绩。大厅 → ModifyInfo / Setting / BattleRecord。游戏列表按dfex及在线配置加载，红二十入口存在但模块不在包中。

H5对应：登录 → 大厅 → 创建/加入房间 → 准备 → 牌桌 → 单局记录 → 下一局/大厅。大厅 → 茶馆 → 创建/加入 → 成员与桌子 → 茶馆房间。大厅底栏 → 战绩/设置；管理后台为独立权限页面。浏览器邀请携带room查询参数，登录后再加入。

每个操作均由新Go服务验证。缺失游戏从入口进入证据缺口说明，不能进入其他游戏桌面冒充。
''')
save('docs/UI_REFERENCE.md','''# UI结构参考

证据：Z/views/ClientScene.lua、views/layer/plaza/GameListLayer.lua、battle/CreateConfigView.lua、各游戏GameViewLayer。图片不能可靠解码的部分未作外观结论。

- 原版为横屏；ClientScene背景按最大比例覆盖并做中心偏移。代码含1334宽适配项，不能证明唯一设计分辨率。
- 大厅顶栏：头像/用户资料，RichNotify公告（约x360、顶部-95），右上设置（右边-39、顶部-50）及分享/客服。
- 大厅左侧主操作：加入房间在创房上方（左+280，中线+115/-110）；不是右侧通用卡片Dashboard。
- 右侧游戏列表TableView宽480，高min(160×条目数,471)，上下翻页箭头在约x1050、中线±265。
- 底栏背景高109，中心y54.5。商城/银行/兑换/福利/茶馆或战绩的中心x约130/350/570/790/1010；部分入口同位按分支替换，不同时堆叠。新版本不接入交易，底栏映射战绩/资料/规则/设置/茶馆。
- 创房为弹窗式游戏列表+选项组；CreateConfigView滚动范围约900×550，偏移25。4人跑得快隐藏减牌方式。在线规则配置缺失不能照搬占位checkbox。
- 牌桌：本家在底部，余座环桌；本家暗牌可见、其他手牌背面；中间公共出牌/动作结果，操作区靠本家。跑得快3/4人、斗十四2/4人有独立布局分支；不将所有游戏固定四人。
- 麻将只有两门数牌/红中类别有逻辑证据。斗十四使用21种骨牌与独立红点，不使用普通扑克代替。
- 结算为桌上弹层；原版分数来自服务器。本轮不填假分数。没有实机截图证明背景主色、字体纹理及动画细节。
''')
save('docs/UI_STYLE_GUIDE.md','''# 原创H5视觉约定

结构依照UI_REFERENCE：横屏顶栏、左侧加房/创房、右侧纵向游戏列表、底部茶馆入口、房间弹窗与环桌布局。

颜色：茶绿桌面、暖金按钮、米白文字牌面、深木色框。这是新系统原创设计决策，非已证实的原版精确色值。包内能确认的局部文字色有房卡黄色(243,251,7)、常见字体28；原图无法可靠解码，不宣称像素还原。

用CSS/SVG自行绘制木纹、边框、牌面、头像、Logo。禁止复制原版背景、角色、品牌、专有图集或音效。触控按钮最小44px；横屏安全区；窄竖屏提供旋转提示且保留页面操作。动态效果遵循减少动画设置。

麻将：可读数字+万/筒及红中，新牌视觉清晰。扑克：四花色、A与5..K。骨牌：上下点数和名称/红点并显，只有分析页，不冒充完整牌局。
''')
save('docs/UI_COMPONENT_MAP.md','''# 组件映射

|原版|新组件|数据来源|
|---|---|---|
|ClientScene/GameListLayer|Lobby/TopBar/EntryRail/BottomBar|/api/catalog、/api/me|
|BattleFind/NBattleCreate|JoinDialog/CreateDialog|服务端创建/加入|
|TH_Main/MemberDlg/ManagerDlg|TeaHouse/MemberList/TeaSettings|茶馆API与角色检查|
|GameView/HandCard/OutCard|Table/Seat/PlayingCard/MahjongTile|每玩家裁剪快照|
|GameEnd/GameEndRecord|RoundResult/History|服务器真实记录|
|Setting/ModifyInfo|Settings/Profile|本地偏好/玩家API|
|RichNotify|Notice|系统公告|

原版图集/动画资源的路径索引见UI_RESOURCE_REFERENCES.json；无授权，均不进入frontend/public。新组件完全独立实现。缺失游戏通过RuleStatus显示证据和待补项。
''')
save('docs/RULES_COVERAGE.md','''# 规则覆盖与发布门槛

覆盖口径为18个必填调查字段是否找到至少一条代码/配置证据，**不是完整度、正确率或完成比例**；部分字段只有消息读取和容量，仍需验证。B实机证据=0。

|玩法|字段有证据/18|资料覆盖率|关键缺口|实施状态决策|
|---|---|---|---|---|
|红中断掐勾（候选150）|13/18|72%|名称、牌墙76/72、发牌/优先级/庄家/支付|独立组合分析器；完整联机暂不开放|
|无红中（候选57）|13/18|72%|3人对应关系、断掐勾启用项、支付/续局|独立组合分析器；完整联机暂不开放|
|红二十110|1/18|6%|除菜单外全部|禁开局，显式缺失|
|跑得快76|15/18|83%|计分、3人减牌、强制过牌、后续首出|4人无积分技术验证局；明确待验证|
|斗十四95|13/18|72%|重复数/发牌/偷牌/优先级/支付|独立组合分析器；禁开局|

逐项证据以docs/rules为准。任何带U或I字段不能作为“原游戏已还原”证据。由于原服务端未提供，五种完整验收目前不能成立；允许的后续实现必须保留待验证状态。不能用普通四川麻将、其他红二十/斗十四替代。
''')
save('docs/REUSE_PLAN.md','''# 复用与授权决策

没有发现足够授权依据可直接复用的完整原游戏工程或服务器。恢复工具是分析参考，原游戏Lua/图集/音效/品牌不复制、不打包。已有解码结果只用于观察常量、协议形状与行为。

|类别|决策|
|---|---|
|完整可用产品实现|未发现|
|可迁移行为|大厅分区、房号、茶馆角色/房间、准备/结算流程；独立重写|
|仅参考|客户端规则提示、牌码/类型、UI资源引用/坐标、原二进制协议|
|独立实现|Go业务与WebSocket、持久化、PWA、权限、房间状态机、原创CSS/SVG|
|公开依赖|锁定版本并保留依赖许可证；Go库与React/Vite等只从官方包源获取|

正式产物排除analysis/extracted、原始ZIP、包内配置、用户数据、凭据和开发缓存。
''')
save('docs/ANALYSIS_SUMMARY.md','''# 分析检查点（2026-10-23）

Phase0/1静态调查已保存：8253文件全索引、4嵌套ZIP CRC、366可读Lua、5509函数、181 UI文件引用；五份18字段规则报告、技术/UI/功能/复用资料齐备。没有正式产品源码先于此检查点编写。

结论：这是Cocos2d-x Lua Android反编译交接包，非完整可迁移工程。四种目标有可读客户端模块；红二十只有ID110下载入口。两种客户麻将名称需要映射确认；无红中候选57是三人；红中150四人、两门加红中；跑得快76默认四人40张A及5..K；斗十四95是21种骨牌。

没有原服规则/发牌/结算实现，不能证明五种游戏可按原规则完整复刻。充分搜索后按用户允许进入明确待验证的技术验证实现：完成平台、真实联机房间/茶馆/记录及证据较充分的跑得快4人无积分测试局；其余保留独立规则模块与可验证的组合分析，不冒充完整游戏。红二十缺失不是其他模块替代的理由。

未解决：原游戏实机对照和资源解码、官方规则/服务器配置、四种完整计分。规则覆盖见RULES_COVERAGE。进入工程阶段不代表规则验收通过。
''')
save('docs/IMPLEMENTATION_PLAN.md','''# 实施计划与技术选择

检查点前置文件已落盘后开始Phase2/3。当前缺口决定采用证据门槛，不静默发明玩法。

1. Go服务端，统一v1 JSON WebSocket命令、requestId去重、seq与version冲突检查、快照重连、服务器私密手牌裁剪、心跳/超时。四人跑得快作为标注待验证的无积分测试局；不开放不确定的完整玩法。
2. TypeScript + React + Vite。DOM牌面先实现低开销触控布局，无原图资源，暂不引入通用游戏框架；按需独立游戏组件，必要动画再评估Pixi。用户建议的Go保留，非原生APK。
3. 账号仅使用用户名和密码注册/登录及安全会话Cookie；当前没有密码重置/找回。后端角色；持久化Repository支持本地原子JSON快照，迁移接口独立。首期单写节点，禁止多个进程共享JSON文件。
4. 茶馆/成员/角色/房间/对局/举报与后台。每个操作服务端校验；记录真实结果，不生成假积分。无支付/提现/原版私服调用。
5. 五个规则模块分别导出状态与已确认算法；红二十拒绝开始。麻将胡形与斗十四14点组合可测试，但不宣称完整裁决。
6. Linux Docker/Caddy部署脚本及健康检查，TLS/PWA；PostgreSQL/Redis升级方案和SQL schema，未接通的部分明确待做，不能仅放Compose声称已使用。
7. Go单元/并发/真实HTTP-WS联机完整跑得快局、刷新重连/越权/隐私/幂等，Playwright实际浏览器流程与横屏截图；100/300/500连接压测记录真实结果。

完成门槛：构建成功、实际运行、权限与持久化测试通过；五玩法完整验收仍取决于补充规则资料。未知规则是产品公开状态而不是隐藏TODO。
''')
save('docs/PROGRESS.md','''# 当前进度

2026-10-23：Phase0/1静态分析完成并保存检查点。分析归档完整性、索引、五规则18项、UI181引用、技术/复用报告已写入。Phase2实施计划确定Go+React/Vite，独立原创资源。尚未开始正式产品源码。

已确认四目标模块+红二十菜单；缺失服务端和计分。待验证规则不标完整。下一阶段完成真实平台与证据支持的4人跑得快无积分联机验证，其他玩法有明确开局门槛。
''')
save('docs/NEXT_STEPS.md','''# 下一步

1. 提交分析检查点Git。
2. 安装官方Go工具链并校验SHA，初始化Go/前端依赖。
3. 完成后端、原创UI、规则分析器和部署脚本。
4. 实际构建/测试、浏览器截图、完整技术验证局、100/300/500连接压测。
5. 补齐原规则后再开放其余游戏和积分；当前无法凭包内资料越过此项。
''')
save('docs/KNOWN_ISSUES.md','''# 已知问题

- 五游戏完整复刻受规则缺口限制，详见RULES_COVERAGE；红二十完全缺模块。
- 客户“断掐勾”与“断勾卡”映射未实机确认。57为三人，不得当四人。
- 服务器结算/发牌/部分选项全部不在包内；客户端结果读取不能逆推出完整规则。
- 大部分PNG不能直接解码；原色调和复杂场景未视觉验证。新资源原创，不称像素还原。
- Go暂未安装；Docker daemon未启动；移动端真机与公网Linux/TLS暂无环境。
''')
save('docs/TEST_REPORT.md','''# 真实测试记录

## 分析阶段

- 原ZIP CRC8253文件通过，路径检查通过，SHA已记录。
- 4个嵌套ZIP CRC通过；两组内容重复；未见红二十模块。
- 安全静态索引运行成功：5509函数、181 UI文件引用。
- 未执行原APK或Lua，未连接原私有接口；原玩法行为未实机验证。

工程测试尚未运行。不能将原解码器的字节码校验测试计作新产品规则测试。
''')
save('docs/DECISIONS.md','''# 决策记录

2026-10-23：分析先行。客户名称按候选映射，不擅自等同；无红中57三人，红中150四人；红二十缺模块。

2026-10-23：正式资源独立绘制，原专有代码/资产不复制。Go+React/Vite本地独立部署，原生Cocos无法直接迁移到H5。首期Repository可迁移JSON单节点，SQL/Redis为后续实施而非已完成依赖。

2026-10-23：技术验证局不赋予未确认积分。缺规则的完整开局关闭；已确认的规则函数可以单独实现与测试。任何人工允许过牌、重复首局先手等都需标为实验假设。
''')
save('AGENTS.md','''# 浩家园持续工作

项目根C:/Users/xinha/Downloads/haojiayuan。先读docs/PROGRESS、NEXT_STEPS、KNOWN_ISSUES、DECISIONS、TEST_REPORT及analysis/USER_REQUEST。

Phase0/1静态分析检查点2026-10-23已完成；原ZIP不修改，提取内容仅分析、不打包、不运行未知程序。原私有端点/凭据不得使用或导出。原游戏代码/图集/音效无授权，仅参考行为。

五种完整规则尚不足；docs/rules和RULES_COVERAGE为证据真源。不得猜测、替换游戏或把待验证玩法标完整。所有五游戏必须独立模块。允许已确认算法和明确标注的技术验证，缺失的完整开局保持关闭。

后续采用Go服务端、React/TypeScript/Vite。持久化单写本地Repository；后续迁移PG/Redis要真实实现后才宣称。每阶段Git提交与真实测试结果，不能伪造完整验收。无真实支付或现金积分。
''')
print('Analysis checkpoint reports written:',len(games),'game reports and platform reports')

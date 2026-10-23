# 红中断掐勾麻将

证据口径：C=可读客户端代码；R=配置/资源；B=实际运行原客户端；I=推断；U=缺失。未运行原客户端，所有 B 均为未验证。静态反编译可能有误，不能等同服务器源码。证据根目录为 `analysis/extracted/majiang-readable-handoff/work/decoded/`。下文 G=game/game，Z=plaza/plaza/src。原包仅内部参考，不进入产品。

## 1. 游戏名称及内部 ID

R：候选对应 ID 150、sparrowhzdgk，包内名“红中断勾卡”。客户“掐”与原包“卡”是否同义待确认。

## 2. 玩家人数

C：PLAYER_COUNT=4，GAME_PLAYER=4。

## 3. 使用的牌

C：两门数字牌 0x11..0x19、0x21..0x29；红中 0x35 为癞子。不是三门四川麻将。

## 4. 牌组数量

C/I：CARD_COUNT=72+4；18种数字各4张+4红中=76。MAX_REPERTORY=72 与此冲突，需服务器验证。

## 5. 初始牌数

C：MAX_COUNT=11；胡牌分析接收2、5、8、11张，三组面子加一对的结构。开局庄闲数量仅客户端容量/消息可见，未有服务器发牌实现。

## 6. 发牌方式

U：服务器给牌；洗牌、补牌、牌墙顺序与分包方式缺失。

## 7. 座位及轮次

C：四座，当前用户/提供者由服务器消息指定；完整动作优先级 U。

## 8. 庄家规则

U：服务器发送庄家，连庄、换庄及首庄选取不在包内。

## 9. 可执行动作

C：出牌、碰、杠、加杠、报胡、胡/点炮及托管消息；红中不可碰杠由 GameView 文本确认。吃牌枚举存在不代表房间实际允许。

## 10. 特殊牌型

C：平胡、对对、清一色、五对/龙五对、将/清组合、带幺、断幺、天/地、杠开、杠炮、抢杠、海底、报叫等标志；是否全部启用依赖服务器。

## 11. 合法性判断

C：GameLogic AnalyseCard、AnalyseBao、SimpleAnalyseChiHu 等；红中作为替牌。不能据此确认所有过胡/抢杠边界。

## 12. 游戏结束条件

C：HU_RESULT、GAME_END 由服务器；U：血战续局、牌墙耗尽及离线终局细则。

## 13. 计分和结算

U：结果包含服务器分数；客户端番提示不是完整付款表。掐/根是否加番存在注释和禁用代码，不能激活。

## 14. 房间可选规则

R/C：NBattleCreateLayer 从服务器取得 optionInfo/specialInfo；不含完整实际选项表。

## 15. 局数设置

U：通用房间字段支持局数，具体允许值从在线配置获取，包内未固定。

## 16. 特殊玩法配置

C：cbMagicCard=0x35、cbMagicIndex=28；报胡与清胡等操作位存在，互斥和优先级 U。

## 17. 原包实现证据

Z/models/dfex.lua 游戏表 ID150；G/chess/sparrowhzdgk/src/models/CMD_Game.lua；同模块 GameLogic.lua、GameClientEngine.lua、views/GameViewLayer.lua（约527行说明红中）。GameLogic AnalyseBao约410、AnalyseCard约485、SimpleAnalyseChiHu约993。

## 18. 尚未确定的问题

名称映射、76/72矛盾、完整发牌、庄家、动作优先级、断掐勾语义、完整结算、选项表和实机行为。

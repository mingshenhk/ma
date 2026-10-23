# 不带红中断掐勾麻将

证据口径：C=可读客户端代码；R=配置/资源；B=实际运行原客户端；I=推断；U=缺失。未运行原客户端，所有 B 均为未验证。静态反编译可能有误，不能等同服务器源码。证据根目录为 `analysis/extracted/majiang-readable-handoff/work/decoded/`。下文 G=game/game，Z=plaza/plaza/src。原包仅内部参考，不进入产品。

## 1. 游戏名称及内部 ID

R：候选 ID57 sparrowsclm“断勾卡血战麻将”；ID114 sparrowerdgk 是二人断勾卡，不能自动代替客户玩法。

## 2. 玩家人数

C：sparrowsclm PLAYER_COUNT=3；GAME_PLAYER=4 仅协议容量。114为2人。

## 3. 使用的牌

C：0x11..0x19、0x21..0x29，不含0x35红中。

## 4. 牌组数量

C/I：18种×4=72，MAX_REPERTORY=72。

## 5. 初始牌数

C：MAX_COUNT=11、MAX_WEAVE=3，胡牌为三组加一对。庄闲实发数量 U。

## 6. 发牌方式

U：服务器负责洗牌与发牌；本地 RandCardList 不能证明线上发牌策略。

## 7. 座位及轮次

C：三人逻辑和四槽协议分离；实际换手由服务器。

## 8. 庄家规则

U：首庄、连庄、胡后换庄均缺失。

## 9. 可执行动作

C：出牌、碰、杠、加杠、报胡、胡、托管；吃牌仅枚举不足以确认开放。

## 10. 特殊牌型

C：平胡、对对、清一色、五对/龙五对、将对、带幺等；不能套用13张七对。

## 11. 合法性判断

C：AnalyseChiHuCard约678、AnalyseWuDui约777、AnalyseQing约823；GetQiaCount约111与 GetGengCount约139。

## 12. 游戏结束条件

U：服务器结束消息和续局消息存在，但完整结束条件缺失。

## 13. 计分和结算

C：GetHuRank约263..400提示平胡1、对对2、清一色/五对3、清对/龙五对4、带幺/清五对5，若干组合6；抢杠/杠开/杠炮/海底/断幺各加1、报叫加2。U：付款方、倍率、封顶和最终支付。根/掐加番调用被注释。

## 14. 房间可选规则

U：每种房间实际 optionInfo 从服务端下发；通用配置不能作为该玩法选项。

## 15. 局数设置

U：具体可选局数缺失。

## 16. 特殊玩法配置

C：掐的客户端函数围绕5点中张；非自摸且当前牌非5返回0；根统计手牌/面子四张。此为函数语义，不代表当前结算启用。

## 17. 原包实现证据

Z/models/dfex.lua ID57、114；G/chess/sparrowsclm/src/models/CMD_Game.lua、GameLogic.lua 及 GameClientEngine.lua。

## 18. 尚未确定的问题

客户是否选择3人57还是2人114、名称映射、完整结算/庄家/操作顺序及选项。

# 组件映射

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

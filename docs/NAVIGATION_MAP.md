# 导航关系

原包：启动/热更新 → Logon → ClientScene → GameListLayer。大厅 → BattleFind（加房）/NBattleCreate（创房）→ GameClientEngine → GameView → GameEnd → 房间/大厅。大厅 → TeaHouse → TH_Main → 桌/成员/管理/战绩。大厅 → ModifyInfo / Setting / BattleRecord。游戏列表按dfex及在线配置加载，红二十入口存在但模块不在包中。

H5对应：登录 → 大厅 → 创建/加入房间 → 准备 → 牌桌 → 单局记录 → 下一局/大厅。大厅 → 茶馆 → 创建/加入 → 成员与桌子 → 茶馆房间。大厅底栏 → 战绩/设置；管理后台为独立权限页面。浏览器邀请携带room查询参数，登录后再加入。

每个操作均由新Go服务验证。缺失游戏从入口进入证据缺口说明，不能进入其他游戏桌面冒充。

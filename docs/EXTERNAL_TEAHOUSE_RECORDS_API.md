# 茶馆战绩外部只读 API

更新：2026-10-25。接口供其他服务（例如由客户或接入方另行开发的微信小程序）读取茶馆战绩。本项目提供 API，不包含微信小程序页面、登录或发布工作。

## 访问方式

- 这是公开只读接口，不要求登录、Cookie、Bearer Token 或其他密钥。知道 `teaHouseId` 的调用方即可读取该茶馆保留期内的战绩。
- 只提供 `GET` 查询。跨域调用允许 `GET` 和 `OPTIONS`；普通 GET 无需自定义请求头。
- 接口只返回最近7天内已结算的小局摘要，不返回当前手牌或实时牌局状态。微信小程序由接入方开发，并需在小程序后台配置部署服务的 HTTPS 合法请求域名。

## 请求

```http
GET /api/v1/teahouses/{teaHouseId}/records?date=today&memberId={userId}
```

线上地址与当前服务同源，例如 `https://game.example.com/api/v1/teahouses/{teaHouseId}/records`。页面「茶馆设置」也会显示当前服务地址。

可选查询参数：

| 参数 | 默认值 | 范围/含义 |
| --- | --- | --- |
| `date` | 无 | 不传时查询最近7天；`today` 查询中国标准时间（UTC+8）今天；也可传 `YYYY-MM-DD` 查询指定日期。指定日期仍受最近7天保留期约束，早于保留期的日期返回空结果，未来日期返回 `400`。 |
| `memberId` | 无 | 按房间保存的玩家列表筛选；与 `date` 组合即可查询某成员当天所在房间的当天对局。 |
| `limit` | `50` | 每页房间数，1–100。 |
| `offset` | `0` | 跳过的房间数，0–10000。 |
| `game` | 无 | 按房间配置中的游戏 key 精确筛选。 |

示例：

```text
/api/v1/teahouses/{teaHouseId}/records?date=today
/api/v1/teahouses/{teaHouseId}/records?date=today&memberId={userId}
/api/v1/teahouses/{teaHouseId}/records?date=2026-10-25&memberId={userId}&game=hongzhong
```

结果按房间最近一局结算时间从新到旧排列；`total` 是筛选后的房间数，不是小局总数。一个房间结果只包含匹配所选日期且仍在保留期内的小局；省略 `date` 时包含最近7天的小局。成员筛选确定房间范围，结果中的 `players` 会保留房间玩家信息，`scores` 按座位顺序排列。

## 响应

成功时返回 JSON：

```json
{
  "teaHouse": { "id": "club-id", "name": "周末茶馆" },
  "retentionDays": 7,
  "generatedAt": 1792890000,
  "total": 1,
  "limit": 50,
  "offset": 0,
  "records": [
    {
      "room": {
        "id": "room-id",
        "config": { "game": "hongzhong" },
        "status": "finished",
        "players": 2,
        "capacity": 2,
        "round": 4,
        "clubId": "club-id",
        "templateId": "template-id"
      },
      "players": [
        { "userId": "user-1", "name": "玩家甲", "avatar": 1, "ready": false, "total": 12 }
      ],
      "rounds": [
        {
          "number": 4,
          "end": "胡牌",
          "scores": [12, -12],
          "wins": [],
          "ledger": [],
          "at": 1792886400
        }
      ],
      "updated": 1792886400
    }
  ]
}
```

`room.config` 是房间配置，`room.status` 为房间状态。玩家字段来自该房间记录。每个小局的 `scores` 为座位顺序的积分结果；`wins` 是玩法结算赢家明细，`ledger` 是转分明细；具体对象字段随游戏规则结算而异。`rounds[].at` 与顶层 `generatedAt`、`updated` 使用 Unix 秒时间戳。

常见错误：茶馆不存在返回 `404`；不支持的方法返回 `405`；日期格式/未来日期或分页参数越界返回 `400`。错误响应为 `{ "error": "..." }`。

## 保留期限与范围

- 页面与 API 均依据 `Round.At` 的 Unix 秒筛选最近7天；指定自然日按中国标准时间（UTC+8）切分。未来时间戳和没有有效结算时间的旧记录不显示。
- 服务启动时清理过期茶馆小局记录，运行中每小时再次清理并保存。清理遇到存储错误会记录错误并在后续周期重试；即使物理清理等待重试，过期数据也不会由页面或 API 返回。
- 茶馆战绩只保留已结算小局及对应的积分/赢家/转分摘要。它不是完整逐步录像，不包含每个摸牌、出牌或响应动作。
- API 与游戏服务同部署，当前使用本地 JSON 单写 Repository；尚未进行真实公网或微信小程序集成验收。

# 第三方软件说明

本发布包包含下列第三方软件生成或链接的运行产物，其许可证原文保存在 `legal/`。

| 组件 | 本次版本 | 许可文件 | 上游 |
| --- | --- | --- | --- |
| React | 19.2.0 | legal/react-LICENSE.txt（MIT，Meta Platforms, Inc. and affiliates） | https://github.com/facebook/react |
| React DOM | 19.2.0 | legal/react-dom-LICENSE.txt（MIT） | https://github.com/facebook/react |
| Scheduler | 0.27.0 | legal/scheduler-LICENSE.txt（MIT） | https://github.com/facebook/react |
| coder/websocket | 1.8.14 | legal/coder-websocket-LICENSE.txt（ISC，Copyright (c) 2025 Coder） | https://github.com/coder/websocket |
| Go 运行时和标准库 | 1.27.3 | legal/go-LICENSE.txt（BSD 3-Clause）及 go-PATENTS.txt | https://go.dev/ |
| Vite 构建注入的浏览器辅助代码 | 7.3.7 | legal/vite-LICENSE.txt | https://github.com/vitejs/vite |

开发工具和构建依赖由 `frontend/package-lock.json`、`backend/go.mod` 与 `backend/go.sum` 锁定。Node.js、npm、TypeScript、Vite、Go 编译器和 node_modules 不随发布包分发；自行安装或再分发工具时请遵守相应许可证。Docker 基础镜像需单独下载，不包含在 ZIP 中。

原客户端包仅作内部静态规则和交互分析。该发布包没有包含原私有接口、账号凭据、原游戏代码、图集或音效。此说明仅记录第三方组件许可，不给本项目自有代码另设开源许可，也不代表取得原客户端素材的再分发授权。

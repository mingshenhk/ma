# 技术识别与环境报告

- 原ZIP 281,750,936字节；SHA-256 `7f1c08d8f1f1843adadfe890c39d1c9052a142064b7d0087fc995f43db63a598`；8253个文件；449,311,036字节解压量；CRC全部通过；路径穿越/加密条目0。完整机器结果见 ARCHIVE_REPORT.json。
- 内容是已有反编译交接包，不是可构建的原工程。Android包名 com.foxuc.scjyproject，Cocos2d-x + Lua，Java启动和SDK壳、JNI原生库；不是Cocos Creator或Unity工程。
- 894个.lua扩展名含压缩/原始字节码；366份可读Lua（大厅167、shell79、棋牌游戏69、休闲51）。DEX2、SO4、Java2144、Smali2216；原生C++源/服务器源未发现。字节码反编译的控制流需核对，不能视为源码正确性证明。
- 1981个.png中仅57有标准PNG头。大部分资源有自定义压缩/加密；19个plist，另有MP3 218、WAV535。图集引用可定位，原始大场景无法直接视觉复现。原包解码工具采用原生SBOX/Huffman机制，但未执行未知工具或APK；不绕过在线鉴权。
- 4个嵌套ZIP为game/plaza两份重复，CRC均通过；清单再次扫描，无红二十模块。不存在CSB/CSD原始布局工程；主要UI由Lua程序化定位。
- 技术证据：work/decoded/plaza/plaza/src、game/game各模块；work/extracted/pkg/AndroidManifest.xml；JADX/Smali/so反编译资料；完整路径和头部摘要见FILE_INVENTORY.csv。
- 工具：Windows/PowerShell，Python3.11.4、Node24.15、npm、git2.54、rg、Java；Docker CLI29.7.2存在，daemon未启动；Go未在PATH。只安装构建/测试需要的官方工具。
- 不运行未知包程序，不连接原服，不导出原包配置中的凭据；产品重新定义协议与品牌资源。

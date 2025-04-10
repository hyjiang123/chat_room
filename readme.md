hello！
本项目是基于 https://github.com/Hacker-Linner/cloud-native-game-server.git 上继续开发，实现了一个聊天室的小功能，仅用于学习交流使用，不作其他用途。
原项目是基于 go 语言开发，其中使用到了 nano 的框架开发；主要采用 websocket 进行双端通信；
本项目在原有基础上，优化了前端界面，改善了使用体验；后端增加在线列表显示，并且将 sessionID 替换成用户可定义的 nickname，提高了用户友好度；此外将聊天数据
提供MySQL数据库进行，使得每次进入聊天室可以看到其他人的聊天记录，完善了功能体验。

项目启动：
1.go 环境的配置
2.MySQL 的环境配置，可根据提供的建表语句； 同时要在 main.go 程序中指定数据库对应的 ip, username, password, databasename信息；
3. go run main.go 启动后端程序
4.访问 demo.html 即可。

本项目框架简单，可以在此基础上再增加功能，包括但不限于：前端页面优化，采用账号密码绑定用户，私聊等。

项目简陋但是是个人小成果，希望能够得到您的喜欢！ 如有问题和意见请联系： huigjiang@163.com 

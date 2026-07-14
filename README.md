# OpportunityOS

OpportunityOS 是一个配置驱动的机会经营核心平台原型。业务对象沿以下链路动态生成：

`机会 -> 孵化项目 -> 业务蓝图 -> 产品 -> SKU -> 执行流程 -> 定价与销售方案`

平台同时管理资源组合、订单计费、履约状态与经营反馈。具体业务类型通过对象定义和关系配置产生，不写死在源码模块中。

## 运行

这是一个无构建依赖的静态应用。可直接打开 `index.html`，或在当前目录运行：

```bash
node dev-server.cjs
```

浏览器中的演示数据会写入 `localStorage`，键名为 `opportunityos-state-v1`。

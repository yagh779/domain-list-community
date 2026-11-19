# domain-list-community（分支）

简体中文 | [English](README.md)

本仓库是
[`v2fly/domain-list-community`](https://github.com/v2fly/domain-list-community)
的精简重新打包分支。它不维护自有域名数据，只重建一个精简的 `dlc.dat`，恰好包含
[`yagh779/sing-geosite`](https://github.com/yagh779/sing-geosite)
中两个 sing-box 规则集所需的数据：

| 条目 | 内容 | 供给标签 |
| --- | --- | --- |
| `cn` | 带 `@!cn` 属性的覆盖域名（来自 `data/cn-overseas`） | `cn@!cn` |
| `geolocation-!cn` | 上游 `geolocation-!cn` 中仅带 `@cn` 属性的域名 | `geolocation-!cn@cn` |

仅发布 `dlc.dat` 与 `dlc.dat.sha256sum`，均位于 `release` 分支。

## 数据源

- `geolocation-!cn@cn`：取自 [`v2fly/domain-list-community`](https://github.com/v2fly/domain-list-community) 最新的 `dlc.dat`。
- `cn@!cn`：取自 [`nekolsd/domain-list-community`](https://github.com/nekolsd/domain-list-community) 最新的 `dlc.dat`，
  并以带 `@!cn` 属性的规则落入 [`data/cn-overseas`](data/cn-overseas)。

## 构建流水线

`Build dlc.dat` 工作流（[.github/workflows/build.yml](.github/workflows/build.yml)）
在每日定时（UTC `30 20 * * *`）以及手动触发时运行：

1. 从两个数据源仓库下载最新的 `dlc.dat`。
2. 为每个目标标签（`cn@!cn`、`geolocation-!cn@cn`）计算稳定的指纹，并与 `sync-state.json` 对比。
3. 当两个指纹均未变化时停止构建（手动强制重建除外）。
4. 当 `cn@!cn` 变化时刷新 `data/cn-overseas`。
5. 重新打包精简的 `dlc.dat`（仅包含上述域名）并连同校验和推送到 `release` 分支。
6. 将更新后的 `sync-state.json` 提交回 `master`。

`sync-state.json` 以标签名为键保存两个指纹：

```json
{
  "cn@!cn": "<sha256>",
  "geolocation-!cn@cn": "<sha256>"
}
```

## 本地用法

```sh
# fingerprints
go run ./cmd/sync hashtag <dlc.dat> 'cn@!cn'
go run ./cmd/sync hashtag <dlc.dat> 'geolocation-!cn@cn'

# refresh the cn@!cn overlay from a nekolsd dlc.dat
go run ./cmd/sync extract-cn-overseas <nekolsd-dlc.dat> data/cn-overseas

# repackage the trimmed dlc.dat
go run ./cmd/build <v2fly-dlc.dat> data/cn-overseas dist/dlc.dat
```

## 仓库结构

| 路径 | 用途 |
| --- | --- |
| `cmd/sync/` | 指纹计算与覆盖数据提取工具 |
| `cmd/build/` | 重新打包精简的 `dlc.dat` |
| `data/cn-overseas` | 供给 `cn@!cn` 的 `@!cn` 覆盖数据 |
| `sync-state.json` | 上次构建的标签指纹 |

上游的 `data/` 目录（上千个分类列表）、上游构建程序及其测试与辅助包均已刻意移除——
本分支只保留重新打包流水线真正需要的文件。

域名规则语法（`domain:`、`full:`、`keyword:`、`regexp:`、`@attr`、`include:`）
遵循上游；详见
[上游 README](https://github.com/v2fly/domain-list-community#structure-of-data)。

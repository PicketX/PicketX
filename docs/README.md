# Design documents

English is the default; Chinese editions carry the same decisions. These documents describe the target architecture, not implemented features. See the [root README](../README.md) for the development scaffold status.

| Document | English | 简体中文 |
| --- | --- | --- |
| Product v0.4.0 | [Product](PRODUCT_DESIGN.md) | [产品](PRODUCT_DESIGN.zh-CN.md) |
| Architecture v0.7.0 | [Architecture](ARCHITECTURE.md) | [架构](ARCHITECTURE.zh-CN.md) |
| Configuration model v0.1.0 | [Model](CONFIGURATION_MODEL.md) | [模型](CONFIGURATION_MODEL.zh-CN.md) |

The product document defines outcomes, the architecture defines system boundaries, and the configuration model refines the agreed object/selection/binding contract. Unresolved schema details are explicitly tracked; illustrative YAML is not an implemented API.

Historical documentation is retained outside this clean source baseline; the previous repository contents remain in Git history. Future behavior changes should record focused ADRs and update both language editions.

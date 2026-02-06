# bugspots-go 機能拡張計画

## 概要

本ドキュメントは、bugspots-go ツールの機能ロードマップを定義する。

## 現状の課題

現在のスコアリングは「変更が多い＝リスクが高い」という仮定に基づいている。この方式には以下の限界がある：

1. **活発に開発中のファイルが常に上位に来る** - 単なる「よく変わるファイル一覧」になりがち
2. **過去のバグとの相関がない** - 実際にバグが発生したファイルとの関連性が考慮されていない
3. **CI連携での実用性が低い** - PR単位での差分分析ができない
4. **重みの根拠が不明確** - ユーザーが適切な重みを判断できない

## 実装済み機能

### ✅ Priority A：JIT基盤機能（実装完了）

以下の機能が `commits` サブコマンドとして実装済み：

- **Change Entropy（変更の散らばり度）**: Shannon entropy による変更分布の定量化
- **Diffusion Metrics**: NF（ファイル数）、ND（ディレクトリ数）、NS（サブシステム数）
- **Size Metrics**: LA（追加行数）、LD（削除行数）
- **JIT Commit Risk Score**: 上記メトリクスの重み付け合計によるリスクスコア

**使用方法**:
```bash
# 基本的な使用
./bugspots-go commits --repo /path/to/repo --since 2025-01-01

# 高リスクコミットのみ表示
./bugspots-go commits --risk-level high

# スコア内訳を表示
./bugspots-go commits --explain

# JSON出力
./bugspots-go commits --format json --output commits.json
```

**実装ファイル**:
- `internal/entropy/shannon.go` - Shannon エントロピー計算
- `internal/aggregation/commit_metrics.go` - コミットメトリクス計算
- `internal/scoring/commit_scorer.go` - JIT コミットスコアリング
- `cmd/commits.go` - commits コマンド
- `internal/output/` - 各種出力フォーマッター

---

### ✅ Priority B：チーム開発の現実に効く機能（実装完了）

#### B1. Ownership / Experience メトリクス強化

**実装内容**:
- `ContributorCommitCounts`: 各コントリビューターのコミット数を追跡
- `OwnershipDispersion`: 所有権の分散度（1 - 最大寄与率）
- スコアリングにおいて、高い分散度（所有権が分散）= 高いリスクとして計算

**計算式**:
```
OwnershipRatio = max(ContributorCommitCounts) / CommitCount
OwnershipDispersion = 1.0 - OwnershipRatio
OwnershipComponent = Ownership Weight × OwnershipDispersion
```

**実装ファイル**:
- `internal/aggregation/file_metrics.go` - ContributorCommitCounts 追跡
- `internal/scoring/file_scorer.go` - Ownership コンポーネント計算

#### B2. Change Coupling（同時変更の結合可視化）

**実装内容**:
- `coupling` サブコマンドとしてファイル間の暗黙的な依存関係を検出
- Jaccard係数、Confidence、Lift による結合強度の定量化
- 大きすぎるコミット（MaxFilesPerCommit超過）はノイズとしてスキップ

**計算式**:
```
Jaccard(A, B) = |A ∩ B| / |A ∪ B|
Confidence = CoCommitCount / FileACommitCount
Lift = P(A,B) / (P(A) × P(B))
```

**使用方法**:
```bash
# 基本的な使用
./bugspots-go coupling --repo /path/to/repo --since 2025-01-01

# 最小共起回数を指定
./bugspots-go coupling --min-co-commits 5

# JSON出力
./bugspots-go coupling --format json --output coupling.json
```

**実装ファイル**:
- `internal/coupling/analyzer.go` - Change Coupling 分析ロジック
- `cmd/coupling.go` - coupling コマンド
- `config/config.go` - CouplingConfig 設定

---

## 現状分析

### 5コンポーネントファイルスコアリング

| コンポーネント | 重み | 説明 |
|---------------|------|------|
| Commit Frequency | 30% | ログ正規化されたコミット数 |
| Code Churn | 25% | 追加行 + 削除行 |
| Recency | 20% | 最終更新からの指数減衰 |
| Burst Activity | 15% | 7日窓での変更集中度 |
| Ownership | 10% | 所有権分散度（1 - 最大寄与率） |

### JIT コミットスコアリング

| コンポーネント | 重み | 説明 |
|---------------|------|------|
| Diffusion | 35% | ファイル数、ディレクトリ数、サブシステム数 |
| Size | 35% | 追加行数 + 削除行数 |
| Entropy | 30% | 変更の Shannon エントロピー |

### アーキテクチャ

```
cmd/                          # CLI コマンド
├── root.go                   # 共通フラグ、App 設定
├── analyze.go                # File Hotspot 分析
├── commits.go                # JIT Commit Risk 分析
└── coupling.go               # Change Coupling 分析

internal/
├── git/
│   ├── reader.go             # Git 履歴読み取り
│   └── models.go             # CommitInfo, FileChange, CommitChangeSet
├── scoring/
│   ├── normalization.go      # NormLog, RecencyDecay
│   ├── file_scorer.go        # 5要素ファイルスコアリング
│   └── commit_scorer.go      # JIT コミットスコアリング
├── aggregation/
│   ├── file_metrics.go       # ファイル単位メトリクス集約
│   └── commit_metrics.go     # コミット単位メトリクス計算
├── burst/
│   └── sliding_window.go     # O(n) バーストスコア計算
├── entropy/
│   └── shannon.go            # Shannon エントロピー計算
├── coupling/
│   └── analyzer.go           # 共変更パターン分析
└── output/
    ├── formatter.go          # 出力インターフェース
    ├── console.go            # テーブル出力
    ├── json.go               # JSON 出力
    ├── csv.go                # CSV 出力
    └── markdown.go           # Markdown 出力

config/
└── config.go                 # 設定構造体、JSON 読み込み
```

---

## 未実装機能

## 優先度A（高）：精度と実用性の根本改善

### A1. バグ修正コミットとの相関を組み込む

**目的**: 過去のバグ修正コミットを特定し、それと相関の高いファイルにスコアを寄せる

**現状の問題**:
現在のスコアは「変更が多い＝リスクが高い」という仮定だけで動いている。これだと活発に開発中のファイルが常に上位に来て、ノイズが大きい。

**実装内容**:
- コミットメッセージからバグ修正パターンを正規表現で抽出
- 「そのファイルが過去にバグ修正された回数」を独立したメトリクスとして追加
- これは元の bugspots アルゴリズムの本質でもある

**設定例**:
```json
{
  "bugPatterns": [
    "\\bfix(ed|es)?\\b",
    "\\bbug\\b",
    "\\bhotfix\\b",
    "\\bpatch\\b",
    "#\\d+",
    "[A-Z]+-\\d+"
  ]
}
```

**CLI オプション**:
```bash
--bug-patterns <REGEX>   バグ修正を示すコミットメッセージパターン
```

**実装予定ファイル**:
- `internal/bugfix/detector.go` - バグ修正コミット検出
- `internal/aggregation/file_metrics.go` - BugfixCount メトリクス追加
- `internal/scoring/file_scorer.go` - Bugfix コンポーネント追加
- `config/config.go` - BugPatterns 設定追加

---

### A2. 差分モード（PR/ブランチ単位の分析）

**目的**: CI連携でPR単位の分析を可能にする

**現状の問題**:
全ファイルを分析するため、CI連携では「今回変更されたファイルの中でリスクが高いものはどれか」を知ることができない。

**実装内容**:
- 2つのコミット/ブランチ間の差分ファイルのみを対象に分析
- 変更されたファイルのリスクスコアを返す
- PR レビューの優先順位付けに直結

**CLI オプション**:
```bash
./bugspots-go analyze --diff origin/main...HEAD
./bugspots-go analyze --diff abc123..def456
```

**出力例**:
```json
{
  "base": "origin/main",
  "head": "HEAD",
  "changedFiles": 5,
  "highRiskFiles": [
    {
      "path": "src/auth/login.go",
      "riskScore": 0.82,
      "changeType": "modified"
    }
  ]
}
```

**CI連携例**:
```yaml
- name: Check PR Risk
  run: |
    ./bugspots-go analyze --diff origin/main...HEAD --format json --output pr-risk.json
    # 高リスクファイルがあれば警告
```

**実装予定ファイル**:
- `internal/git/diff.go` - 差分ファイル取得
- `cmd/analyze.go` - `--diff` オプション追加
- `internal/output/` - 差分モード用出力

---

## 優先度B（中）：精度向上

### B1. ファイル複雑度メトリクスの追加

**目的**: 変更パターンだけでなく、コード自体の特性を見る

**根拠**:
ファイルサイズ（行数）は最も単純で効果的な複雑度の代理指標。大きいファイルほどバグが潜む確率が高いのは実証研究でも裏付けられている。

**実装内容**:
- ファイル行数を取得し、メトリクスに追加
- 将来的には外部ツール連携（サイクロマティック複雑度）も検討

**CLI オプション**:
```bash
--include-complexity   ファイル複雑度をスコアに含める
```

**実装予定ファイル**:
- `internal/complexity/analyzer.go` - ファイル複雑度計算
- `internal/aggregation/file_metrics.go` - FileSize, Complexity メトリクス追加
- `internal/scoring/file_scorer.go` - Complexity コンポーネント追加

---

### B2. スコアキャリブレーション

**目的**: ユーザーが適切な重みを判断できるようにする

**現状の問題**:
重みは固定のデフォルト値で、ユーザーが適切な重みを判断できない。

**実装内容**:
1. **実績ベースのキャリブレーション**: 過去のバグ修正コミット（A1で検出）を「正解データ」として、各メトリクスの重みを最適化（単純な線形回帰）
2. **検出率表示**: `--calibrate` オプションで「この重みでの過去のバグ修正ファイルの検出率（再現率）」を表示

**CLI オプション**:
```bash
./bugspots-go calibrate --repo /path/to/repo --since 2024-01-01
```

**出力例**:
```
Calibration Results (based on 150 bug-fix commits):

Current weights detection rate: 65%

Recommended weights:
  commit:    0.20 (current: 0.30)
  churn:     0.30 (current: 0.25)  ← このリポジトリでは churn の相関が強い
  recency:   0.15 (current: 0.20)
  burst:     0.20 (current: 0.15)
  ownership: 0.05 (current: 0.10)
  bugfix:    0.10 (new)

Expected detection rate with recommended weights: 78%
```

**実装予定ファイル**:
- `internal/calibration/optimizer.go` - 重み最適化
- `cmd/calibrate.go` - calibrate コマンド

---

## 優先度C（低）：運用改善

### C1. トレンド分析

**目的**: スコアの時系列変化を追跡する

**根拠**:
「先月からスコアが急上昇しているファイル」は、「元からスコアが高いが安定しているファイル」より優先してレビューすべき。

**実装内容**:
- 2時点の分析結果を比較
- スコア上昇率でソート

**CLI オプション**:
```bash
./bugspots-go analyze --compare-with previous-report.json
```

**出力例**:
```
Trend Analysis (compared to 2025-01-01):

Rising Risk Files:
  src/auth/login.go      0.45 → 0.82 (+82%)
  src/api/handler.go     0.30 → 0.55 (+83%)

Declining Risk Files:
  src/core/engine.go     0.75 → 0.60 (-20%)
```

**実装予定ファイル**:
- `internal/trend/analyzer.go` - トレンド分析
- `cmd/analyze.go` - `--compare-with` オプション追加

---

### C2. JIT経験メトリクス拡張

**目的**: 既存の `commits` コマンドに経験ベースのメトリクスを追加

**追加メトリクス**:
- `Experience`: 変更者の該当ファイル/領域での過去経験値
- `History`: 変更ファイルの過去バグ履歴

**JIT スコア計算式（拡張版）**:
```
JIT_Score = w1×Diffusion + w2×Size + w3×Entropy +
            w4×Experience + w5×History + w6×BurstContext
```

**実装予定ファイル**:
- `internal/experience/calculator.go` - 経験値計算
- `internal/scoring/commit_scorer.go` - 拡張スコアリング

---

### C3. 重複変更（Duplicate Changes）除去

**目的**: 自動整形やボイラープレート変更によるノイズを除去

**検出方法**:
1. 同一内容の連続コミットを検出
2. コミットメッセージパターンで自動生成を識別
3. 変更内容のハッシュで重複を判定

**CLI オプション**:
```bash
--dedupe             重複変更を除外（デフォルト: off）
--dedupe-patterns    除外パターン定義ファイル
```

**実装予定ファイル**:
- `internal/dedupe/detector.go` - 重複検出ロジック
- `config/config.go` - DedupeConfig 追加

---

### C4. リネーム追跡の強化

**目的**: ファイル名変更を追跡し、同一ファイルとして扱う

**モード**:
- `off`: リネーム追跡なし（デフォルト）
- `simple`: 基本的なリネーム検出
- `aggressive`: 内容ベースの類似度検出

**CLI オプション**:
```bash
--rename-detect off|simple|aggressive
```

**実装予定ファイル**:
- `internal/git/reader.go` - リネーム検出オプション追加
- `internal/git/rename.go` - リネーム追跡ロジック

---

## 優先度D：パフォーマンス最適化

### D1. インクリメンタル分析

**目的**: 大規模リポジトリでの分析時間短縮

**実装内容**:
- 前回分析結果のキャッシュ
- 差分コミットのみの分析
- SQLite または JSON ファイルでの永続化

**CLI オプション**:
```bash
--cache              キャッシュを使用（デフォルト: off）
--cache-dir <PATH>   キャッシュディレクトリ
--refresh            キャッシュを無効化して再分析
```

### D2. 並列処理

**目的**: マルチコアを活用した分析高速化

**実装内容**:
- コミット解析の並列化
- ファイルメトリクス計算の並列化
- goroutine + channel による実装

---

## 設定ファイル拡張

### .bugspots.json 拡張例
```json
{
  "scoring": {
    "halfLifeDays": 30,
    "weights": {
      "commit": 0.20,
      "churn": 0.20,
      "recency": 0.15,
      "burst": 0.15,
      "ownership": 0.10,
      "bugfix": 0.20
    }
  },
  "bugPatterns": [
    "\\bfix(ed|es)?\\b",
    "\\bbug\\b",
    "\\bhotfix\\b",
    "\\bpatch\\b",
    "#\\d+",
    "[A-Z]+-\\d+"
  ],
  "burst": {
    "windowDays": 7
  },
  "commitScoring": {
    "weights": {
      "diffusion": 0.25,
      "size": 0.20,
      "entropy": 0.20,
      "experience": 0.15,
      "history": 0.10,
      "burstContext": 0.10
    },
    "thresholds": {
      "high": 0.7,
      "medium": 0.4
    }
  },
  "coupling": {
    "minCoCommits": 3,
    "minJaccardThreshold": 0.1,
    "maxFilesPerCommit": 50,
    "topPairs": 50
  },
  "complexity": {
    "enabled": false,
    "weight": 0.10
  },
  "filters": {
    "include": ["src/**", "apps/**"],
    "exclude": ["**/vendor/**", "**/testdata/**"]
  },
  "dedupe": {
    "enabled": false,
    "patterns": [
      "^(chore|style|format):",
      "^\\[auto\\]"
    ]
  },
  "cache": {
    "enabled": false,
    "dir": ".bugspots-cache"
  }
}
```

---

## 出力形式拡張

### JSON出力拡張例（Phase 4以降）
```json
{
  "repo": "/path/to/repo",
  "since": "2024-01-01T00:00:00Z",
  "until": "2025-01-15T12:00:00Z",
  "generatedAt": "2025-01-15T12:30:00Z",
  "summary": {
    "totalFiles": 150,
    "analyzedCommits": 523,
    "uniqueContributors": 12
  },
  "items": [
    {
      "rank": 1,
      "path": "src/core/auth.go",
      "riskScore": 0.89,
      "metrics": {
        "commitCount": 45,
        "addedLines": 1200,
        "deletedLines": 800,
        "lastModified": "2025-01-14T15:30:00Z",
        "contributorCount": 5,
        "burstScore": 0.85,
        "ownershipDispersion": 0.55
      },
      "breakdown": {
        "commitComponent": 0.22,
        "churnComponent": 0.18,
        "recencyComponent": 0.14,
        "burstComponent": 0.13,
        "ownershipComponent": 0.07
      },
      "coupling": [
        { "file": "src/core/session.go", "jaccard": 0.78 },
        { "file": "src/api/auth_handler.go", "jaccard": 0.65 }
      ]
    }
  ]
}
```

---

## 優先度まとめ

| 優先度 | 機能 | 理由 |
|--------|------|------|
| **A（高）** | バグ修正コミットの相関 | 精度への直接的インパクトが最大。これがないと「よく変わるファイル一覧」でしかない |
| **A（高）** | 差分モード | CI連携の実用性が劇的に変わる |
| **B（中）** | ファイル複雑度 | 実装コストが低い割に精度が上がる |
| **B（中）** | スコアキャリブレーション | バグ修正相関の実装後にやるとさらに効果的 |
| **C（低）** | トレンド分析 | 運用が定着してからで良い |
| **C（低）** | JIT経験メトリクス | 有用だが実装コストが高め |
| **C（低）** | 重複変更除去 | ノイズ対策として有用 |
| **D** | パフォーマンス最適化 | 大規模リポジトリ対応時 |

**推奨**: A1（バグ修正相関）と A2（差分モード）を先に実装すれば、ツールの価値が根本的に変わる。

---

## 実装フェーズ

### ✅ Phase 1: 基盤移植（完了）
1. ✅ 設定システム（`config/config.go`）
2. ✅ Git Reader 拡張（`internal/git/reader.go`, `models.go`）
3. ✅ 正規化サービス（`internal/scoring/normalization.go`）
4. ✅ ファイルメトリクス集約（`internal/aggregation/file_metrics.go`）
5. ✅ バーストスコア計算（`internal/burst/sliding_window.go`）
6. ✅ ファイルリスクスコアラー（`internal/scoring/file_scorer.go`）

### ✅ Phase 2: JIT コミット分析（完了）
1. ✅ Shannon エントロピー（`internal/entropy/shannon.go`）
2. ✅ コミットメトリクス計算（`internal/aggregation/commit_metrics.go`）
3. ✅ コミットリスクスコアラー（`internal/scoring/commit_scorer.go`）
4. ✅ commits コマンド（`cmd/commits.go`）

### ✅ Phase 3: Change Coupling 分析（完了）
1. ✅ Change Coupling Analyzer（`internal/coupling/analyzer.go`）
2. ✅ coupling コマンド（`cmd/coupling.go`）
3. ✅ 出力形式（Console, JSON, CSV, Markdown）

### Phase 4: 精度と実用性の根本改善（次期優先）
1. バグ修正コミット検出（`internal/bugfix/detector.go`）
2. Bugfix メトリクスのスコアリング統合
3. 差分モード実装（`--diff` オプション）
4. CI連携用出力形式

### Phase 5: 精度向上
1. ファイル複雑度メトリクス追加
2. スコアキャリブレーション（`calibrate` コマンド）
3. トレンド分析（`--compare-with` オプション）

### Phase 6: 運用改善
1. JIT経験メトリクス拡張
2. 重複変更検出ロジック
3. リネーム追跡オプション

### Phase 7: パフォーマンス最適化
1. インクリメンタル分析（キャッシュ）
2. 並列処理実装
3. 大規模リポジトリ対応

---

## 検証方法

### 単体テスト
- 各 Calculator / Analyzer の計算ロジックテスト
- 正規化・重み付けテスト
- エッジケース（空コミット、単一ファイル等）

```bash
go test ./...
```

### 統合テスト
- 実際の Git リポジトリでの E2E テスト
- 出力形式の整合性テスト
- CLI オプションの組み合わせテスト

### 実用性検証
- 既知のバグ履歴を持つ OSS リポジトリで精度評価
- 既存ホットスポットツール（bugspots 等）との比較
- 実務プロジェクトでのレビュー優先度付け検証

---

## 参考文献

1. [Change Bursts as Defect Predictors](https://www.st.cs.uni-saarland.de/publications/files/nagappan-issre-2010.pdf)
2. [An Empirical Study of Just-in-Time Defect Prediction](https://posl.ait.kyushu-u.ac.jp/~kamei/publications/Fukushima_MSR2014.pdf)
3. [Predicting faults using the complexity of code changes](https://dl.acm.org/doi/10.1109/ICSE.2009.5070510)
4. [Effort-aware just-in-time defect identification in practice](https://ink.library.smu.edu.sg/sis_research/6632/)
5. [Co-Change Graph Entropy: A New Process Metric](https://dl.acm.org/doi/10.1145/3756681.3757037)
6. [A Formal Explainer for Just-In-Time Defect Predictions](https://people.eng.unimelb.edu.au/pstuckey/papers/TOSEM.pdf)
7. [Leveraging Fault Localisation to Enhance Defect Prediction](https://posl.ait.kyushu-u.ac.jp/~kamei/publications/Sohn_SANER2021.pdf)
8. [The Impact of Duplicate Changes on Just-in-Time Defect Prediction](https://yanmeng.github.io/papers/TR212.pdf)

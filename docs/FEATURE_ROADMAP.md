# bugspots-go 機能拡張計画

## 概要

本ドキュメントは、後続研究（ホットスポット／プロセスメトリクス／JIT欠陥予測）に基づき、bugspots-go ツールの機能ロードマップを定義する。

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

## 優先度C：JIT拡張機能

### C1. JIT経験メトリクス拡張

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

### C2. AI レビュー向け危険理由出力

**目的**: LLMベースのコードレビューに渡すための構造化データ生成

**出力形式**:
```json
{
  "analysisContext": {
    "commit": "abc123",
    "author": "developer@example.com",
    "timestamp": "2025-01-15T10:30:00Z"
  },
  "riskFactors": [
    {
      "factor": "HIGH_DIFFUSION",
      "severity": "HIGH",
      "description": "変更が複数サブシステムに跨がる",
      "value": 0.85,
      "threshold": 0.6
    },
    {
      "factor": "LOW_EXPERIENCE",
      "severity": "MEDIUM",
      "description": "変更者の該当ファイル経験が少ない",
      "value": 0.23,
      "threshold": 0.4
    }
  ],
  "reviewGuidance": {
    "focusAreas": ["認証ロジックの変更", "APIエンドポイントの整合性"],
    "suggestedReviewers": ["senior-dev@example.com"]
  }
}
```

**実装予定ファイル**:
- `internal/output/ai_review.go` - AI レビュー向け出力
- `cmd/analyze.go` - `--format ai-review` オプション追加

---

## 優先度D：分析精度向上機能

### D1. 重複変更（Duplicate Changes）除去

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

### D2. リネーム追跡の強化

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

## 優先度E：パフォーマンス最適化

### E1. インクリメンタル分析

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

### E2. 並列処理

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
      "commit": 0.25,
      "churn": 0.20,
      "recency": 0.15,
      "burst": 0.15,
      "ownership": 0.10,
      "entropy": 0.15
    }
  },
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

### Phase 4: AI連携・重複除去（未実装）
1. AI連携用JSON出力形式実装
2. 重複変更検出ロジック実装
3. `--dedupe` オプション追加
4. パターンマッチング設定機能

### Phase 5: 経験メトリクス・リネーム追跡（未実装）
1. 経験値計算ロジック実装
2. リネーム追跡オプション実装
3. JIT スコアリング拡張

### Phase 6: パフォーマンス最適化（未実装）
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

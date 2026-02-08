# bugspots-go 機能拡張計画

## 概要

本ドキュメントは、bugspots-go ツールの機能ロードマップを定義する。

## 実装済み機能

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

### ✅ Phase 4: 精度と実用性の根本改善（完了）
1. ✅ バグ修正コミット検出（`internal/bugfix/detector.go`）
2. ✅ Bugfix メトリクスの6要素スコアリング統合
3. ✅ 差分モード実装（`--diff` オプション）
4. ✅ CI連携（`--ci-threshold` オプション、CI/NDJSON出力形式）

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

#### B3. バグ修正コミットとの相関

**実装内容**:
- コミットメッセージからバグ修正パターンを正規表現で抽出（大文字小文字を区別しない）
- 「そのファイルが過去にバグ修正された回数」を6要素スコアリングの1コンポーネントとして統合
- 設定ファイルおよび `--bug-patterns` CLI フラグでパターンをカスタマイズ可能

**デフォルトパターン**:
```
\bfix(ed|es)?\b
\bbug\b
\bhotfix\b
\bpatch\b
```

**使用方法**:
```bash
# デフォルトパターンで分析
./bugspots-go analyze --repo /path/to/repo

# カスタムパターンを指定
./bugspots-go analyze --repo /path/to/repo --bug-patterns "\\bfix(ed|es)?\\b" --bug-patterns "\\bbug\\b"
```

**実装ファイル**:
- `internal/bugfix/detector.go` - バグ修正コミット検出
- `internal/aggregation/file_metrics.go` - BugfixCount メトリクス
- `internal/scoring/file_scorer.go` - Bugfix コンポーネント（重み 0.20）
- `config/config.go` - BugfixConfig 設定

#### B4. 差分モード（PR/ブランチ単位の分析）

**実装内容**:
- 2つのコミット/ブランチ間の差分ファイルのみを対象に分析
- `--diff` フラグで三点（`...`）および二点（`..`）構文をサポート
- リネームも正しく追跡（OldPath と Path の両方を含む）
- `--ci-threshold` によるCI自動ゲーティング

**使用方法**:
```bash
# PRの変更ファイルのみ分析
./bugspots-go analyze --diff origin/main...HEAD

# CIでしきい値超過を検出
./bugspots-go analyze --diff origin/main...HEAD --ci-threshold 0.7
```

**CI連携例**:
```yaml
- name: Check PR Risk
  run: |
    ./bugspots-go analyze --diff origin/main...HEAD --ci-threshold 0.7
    # しきい値超過で非ゼロ終了
```

**実装ファイル**:
- `internal/git/diff.go` - 差分ファイル取得（`git diff --name-status -z`）
- `cmd/analyze.go` - `--diff` / `--ci-threshold` オプション

#### B5. リネーム追跡

**実装内容**:
- 3つの検出モードをサポート
  - `off`: リネーム追跡なし（`--no-renames`）
  - `simple`: 完全一致のリネーム検出（`-M100%`、デフォルト）
  - `aggressive`: 類似度ベースのリネーム検出（`-M60%`）

**使用方法**:
```bash
./bugspots-go analyze --repo /path/to/repo --rename-detect aggressive
./bugspots-go analyze --repo /path/to/repo --rename-detect off
```

**実装ファイル**:
- `internal/git/reader_gitcli.go` - リネーム検出モード実装
- `cmd/context.go` - `parseRenameDetectFlag()`

---

## 現状分析

### 6コンポーネントファイルスコアリング

| コンポーネント | 重み | 説明 |
|---------------|------|------|
| Commit Frequency | 25% | ログ正規化されたコミット数 |
| Code Churn | 20% | 追加行 + 削除行 |
| Bugfix | 20% | ログ正規化されたバグ修正コミット数 |
| Recency | 15% | 最終更新からの指数減衰（半減期30日） |
| Burst Activity | 10% | 7日窓での変更集中度 |
| Ownership | 10% | 所有権分散度（1 - 最大寄与率） |

### JIT コミットスコアリング

| コンポーネント | 重み | 説明 |
|---------------|------|------|
| Diffusion | 35% | ファイル数、ディレクトリ数、サブシステム数 |
| Size | 35% | 追加行数 + 削除行数 |
| Entropy | 30% | 変更の Shannon エントロピー |

### 出力形式

| 形式 | フラグ | 説明 |
|------|--------|------|
| console | `--format console` | カラー付きテーブル出力（デフォルト） |
| json | `--format json` | 標準 JSON |
| csv | `--format csv` | CSV スプレッドシート |
| markdown | `--format markdown` | Markdown テーブル |
| ci | `--format ci` | NDJSON（1行1JSONオブジェクト、CI パイプライン向け） |

### アーキテクチャ

```
app.go                        # エントリポイント

cmd/                          # CLI コマンド
├── root.go                   # 共通フラグ、App 設定、出力形式解析
├── context.go                # CommandContext（共通セットアップ）
├── analyze.go                # 6要素 File Hotspot 分析
├── commits.go                # JIT Commit Risk 分析
├── coupling.go               # Change Coupling 分析
└── calibrate.go              # スコア重みキャリブレーション

internal/
├── git/
│   ├── reader.go             # HistoryReader（Git 履歴読み取り）
│   ├── reader_gitcli.go      # Git CLI 実装（numstat/raw 出力解析）
│   ├── models.go             # CommitChangeSet, FileChange, ReadOptions
│   └── diff.go               # 差分ファイル取得（--diff 用）
├── scoring/
│   ├── normalization.go      # NormLog, RecencyDecay, Clamp
│   ├── file_scorer.go        # 6要素ファイルリスクスコアリング
│   └── commit_scorer.go      # JIT コミットリスクスコアリング
├── aggregation/
│   ├── file_metrics.go       # ファイル単位メトリクス集約
│   └── commit_metrics.go     # コミット単位メトリクス計算
├── bugfix/
│   └── detector.go           # バグ修正コミット検出（正規表現）
├── burst/
│   └── sliding_window.go     # O(n) バーストスコア計算
├── entropy/
│   └── shannon.go            # Shannon エントロピー計算
├── coupling/
│   └── analyzer.go           # 共変更パターン分析（Jaccard）
└── output/
    ├── formatter.go          # 出力インターフェース、共通ヘルパー
    ├── console.go            # カラー付きテーブル出力
    ├── json.go               # JSON 出力
    ├── csv.go                # CSV 出力
    ├── markdown.go           # Markdown 出力
    └── ci.go                 # NDJSON 出力（CI パイプライン用）

config/
└── config.go                 # 設定構造体、JSON 読み込み、デフォルト値
```

---

## 実装済み機能（追加）

### ✅ 優先度A（高）：精度向上

#### ✅ A1. ファイル複雑度メトリクスの追加

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

**実装ファイル**:
- `internal/complexity/analyzer.go` - ファイル複雑度計算（git cat-file --batch による行数カウント）
- `internal/aggregation/file_metrics.go` - FileSize メトリクス追加
- `internal/scoring/file_scorer.go` - Complexity コンポーネント追加（7要素スコアリング）
- `config/config.go` - WeightConfig に Complexity を追加
- `cmd/analyze.go` - `--include-complexity` フラグ追加
- 出力ファイル群（console, json, csv, markdown）に Lines/Cx カラム追加

---

#### ✅ A2. スコアキャリブレーション

**目的**: ユーザーが適切な重みを判断できるようにする

**現状の問題**:
重みは固定のデフォルト値で、ユーザーが適切な重みを判断できない。

**実装内容**:
1. **実績ベースのキャリブレーション**: 過去のバグ修正コミット（検出済み）を「正解データ」として、各メトリクスの重みを最適化（単純な線形回帰）
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
  commit:    0.20 (current: 0.25)
  churn:     0.25 (current: 0.20)  ← このリポジトリでは churn の相関が強い
  recency:   0.10 (current: 0.15)
  burst:     0.15 (current: 0.10)
  ownership: 0.05 (current: 0.10)
  bugfix:    0.25 (current: 0.20)

Expected detection rate with recommended weights: 78%
```

**実装ファイル**:
- `internal/calibration/optimizer.go` - 座標降下法による重み最適化
- `cmd/calibrate.go` - calibrate サブコマンド
- `cmd/root.go` - コマンド登録

---

### 未実装機能

### 優先度B（中）：運用改善

#### B1. トレンド分析

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

#### B2. JIT経験メトリクス拡張

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

#### B3. 重複変更（Duplicate Changes）除去

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

### 優先度C（低）：パフォーマンス最適化

#### C1. インクリメンタル分析

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

#### C2. 並列処理

**目的**: マルチコアを活用した分析高速化

**実装内容**:
- コミット解析の並列化
- ファイルメトリクス計算の並列化
- goroutine + channel による実装

---

## 設定ファイル

### .bugspots.json 現在の構成

```json
{
  "scoring": {
    "halfLifeDays": 30,
    "weights": {
      "commit": 0.25,
      "churn": 0.20,
      "recency": 0.15,
      "burst": 0.10,
      "ownership": 0.10,
      "bugfix": 0.20
    }
  },
  "bugfix": {
    "patterns": [
      "\\bfix(ed|es)?\\b",
      "\\bbug\\b",
      "\\bhotfix\\b",
      "\\bpatch\\b"
    ]
  },
  "burst": {
    "windowDays": 7
  },
  "commitScoring": {
    "weights": {
      "diffusion": 0.35,
      "size": 0.35,
      "entropy": 0.30
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
}
```

### 拡張予定の設定項目

```json
{
  "complexity": {
    "enabled": false,
    "weight": 0.10
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

## 優先度まとめ

| 優先度 | 機能 | 理由 |
|--------|------|------|
| **A（高）** | ファイル複雑度 | 実装コストが低い割に精度が上がる |
| **A（高）** | スコアキャリブレーション | 検出済みバグ修正データを正解として重み最適化が可能 |
| **B（中）** | トレンド分析 | 運用が定着してからで良い |
| **B（中）** | JIT経験メトリクス | 有用だが実装コストが高め |
| **B（中）** | 重複変更除去 | ノイズ対策として有用 |
| **C（低）** | パフォーマンス最適化 | 大規模リポジトリ対応時 |

---

## 実装フェーズ

### Phase 5: 精度向上（次期優先）
1. ファイル複雑度メトリクス追加
2. スコアキャリブレーション（`calibrate` コマンド）

### Phase 6: 運用改善
1. トレンド分析（`--compare-with` オプション）
2. JIT経験メトリクス拡張
3. 重複変更検出ロジック

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

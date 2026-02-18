# scribe2srt

將音訊/影片檔案轉換為 SRT 字幕的命令列工具。使用 ElevenLabs Scribe v2 語音辨識 API 進行轉錄，並透過智慧分句與合併演算法產出符合專業品質的字幕檔案。

## 功能特色

- **多種媒體格式支援** — 音訊（`.mp3`、`.m4a`、`.wav`、`.flac`、`.ogg`、`.aac`）及影片（`.mp4`、`.mov`、`.mkv`、`.avi`、`.flv`、`.webm`）
- **CJK 語系最佳化** — 針對中日韓文與拉丁語系分別設定字元速率（CPS）與每行字數（CPL）預設值
- **智慧字幕處理** — 三階段處理流程：前處理 → 分句 → 合併，以三層級標點符號優先權系統進行分句
- **長音訊自動分段** — 超過 90 分鐘的音訊自動切割並行處理
- **並行上傳** — 透過 errgroup 實現有限並行度的分段上傳，搭配速率限制與指數退避重試
- **音訊事件標記** — 可標記音樂、笑聲等非語音事件

## 系統需求

- **Go** 1.25.0 或以上
- **ffmpeg** 與 **ffprobe** — 用於影片音訊擷取及長音訊分段（需在 `PATH` 中可用）

## 安裝

```bash
git clone https://github.com/your-username/scribe2srt-cli.git
cd scribe2srt-cli
make build
```

編譯後的執行檔為 `scribe2srt`。

## 使用方式

### 轉錄

將音訊或影片檔轉錄為 SRT 字幕檔：

```bash
# 基本用法（自動偵測語言）
scribe2srt transcribe input.mp4

# 指定語言
scribe2srt transcribe input.mp4 -l ja

# 指定輸出檔路徑
scribe2srt transcribe input.mp4 -l zh -o output.srt

# 詳細日誌輸出
scribe2srt transcribe input.mp4 -l ko -v
```

#### 支援語言

| 代碼   | 語言   |
|--------|--------|
| `auto` | 自動偵測 |
| `zh`   | 中文   |
| `ja`   | 日文   |
| `ko`   | 韓文   |
| `en`   | 英文   |

#### 轉錄選項

| 旗標 | 縮寫 | 預設值 | 說明 |
|------|------|--------|------|
| `--language` | `-l` | `auto` | 語言代碼 |
| `--output` | `-o` | `<輸入檔>.srt` | 輸出 SRT 檔路徑 |
| `--tag-audio-events` | | `true` | 標記音訊事件 |
| `--no-async` | | `false` | 停用並行處理 |
| `--max-concurrent` | `-j` | `3` | 最大並行上傳數 |
| `--max-retries` | | `3` | 每段最大重試次數 |
| `--rate-limit` | | `30` | 每分鐘 API 請求上限 |
| `--split-duration` | | `90` | 音訊分段門檻（分鐘） |
| `--save-json` | | `false` | 同時儲存轉錄 JSON |

#### 字幕參數調整

| 旗標 | 預設值（CJK / 拉丁） | 說明 |
|------|----------------------|------|
| `--min-duration` | `0.83` 秒 | 字幕最短時長 |
| `--max-duration` | `12.0` 秒 | 字幕最長時長 |
| `--min-gap` | `0.083` 秒 | 字幕間最小間距 |
| `--cjk-cps` | `11` | CJK 每秒字元數上限 |
| `--latin-cps` | `15` | 拉丁語系每秒字元數上限 |
| `--cjk-cpl` | `25` | CJK 每行字元數上限 |
| `--latin-cpl` | `42` | 拉丁語系每行字元數上限 |

### 全域選項

| 旗標 | 縮寫 | 說明 |
|------|------|------|
| `--verbose` | `-v` | 顯示詳細日誌（DEBUG 層級） |
| `--quiet` | `-q` | 僅顯示錯誤訊息 |

## 處理流程

```
輸入檔案
  → [ffmpeg] 從影片擷取音訊，超過 90 分鐘則分段
  → [worker] 處理各分段（並行或循序）
      → [api] 上傳至 ElevenLabs STT，含重試與速率限制
  → [pipeline] 三階段字幕處理：
      階段 0：PreprocessWords — 分離音訊事件、合併空白與 CJK 標點
      階段 1：SentenceSplitter — 依標點優先權分句
      階段 2：IntelligentMerger — 貪婪合併 + 後處理最佳化
  → 輸出 .srt 字幕檔
```

## 開發

```bash
make build                              # 編譯
make run ARGS="transcribe input.mp4 -l ja"  # 直接執行（不產出二進位檔）
make test                               # 執行測試
make clean                              # 清除編譯產物
```

執行單一測試：

```bash
go test ./internal/pipeline/ -run TestName
```

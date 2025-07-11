
# 📑 Document Processor with Priority Queue and Worker Pattern (Go)

A concurrent document processing system in Go using a configurable worker pattern, priority queue, and graceful shutdown handling.

## 📌 Features

- ✅ Queue-based processing to handle high-volume document jobs
- ✅ Priority queue: urgent applications processed first
- ✅ Configurable number of workers based on system load
- ✅ Graceful shutdown handling to prevent data loss
- ✅ Simple callback function for result handling

---

## 📦 Project Structure

```
.
├── main.go
├── processor.go
└── README.md
```

---

## ⚙️ How It Works

- Jobs are pushed into a **priority queue** based on their `Priority` value.
- Multiple workers run concurrently to fetch and process jobs from the queue.
- Jobs with higher priority are processed first.
- Workers can be gracefully shut down using a `context.Context`.
- Each job has a `Callback` function that is called after the job is processed.

---

## 🚀 How to Run

1. **Clone this repo**

```bash
git clone https://github.com/yourusername/document-processor-go.git
cd document-processor-go
```

2. **Run the application**

```bash
go run main.go
```

---

## 📝 Example Output

```text
Worker 1 processing: APP-2
Worker 2 processing: APP-1
Worker 0 processing: APP-3
Result: Processed successfully
...
Worker 2 stopped
Worker 0 stopped
Worker 1 stopped
```

---

## 🧰 Configuration

- Configure the number of workers when initializing:

```go
processor := NewDocumentProcessor(3)
```

---

## 📊 Priority Queue Behavior

- Higher `Priority` value means higher priority.
- Jobs with priority `5` will be processed before those with `3`, and so on.

---

## 📦 Dependencies

- Standard Go libraries:
  - `container/heap`
  - `context`
  - `sync`
  - `time`
  - `fmt`
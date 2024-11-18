// Copyright (c) 2024 RoseLoverX

package telegram

import (
	"bytes"
	"context"
	"crypto/md5"
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
)

const (
	DEFAULT_WORKERS = 4
	DEFAULT_PARTS   = 512 * 512
)

type UploadOptions struct {
	// Worker count for upload file.
	Threads int `json:"threads,omitempty"`
	//  Chunk size for upload file.
	ChunkSize int32 `json:"chunk_size,omitempty"`
	// File name for upload file.
	FileName string `json:"file_name,omitempty"`
	// output Callback for upload progress, total parts and uploaded parts.
	ProgressCallback func(totalParts int64, uploadedParts int64) `json:"-"`
}

type Sender struct {
	c *Client // Holds client information
}

type WorkerPool struct {
	sync.Mutex
	workers []*Sender    // List of all workers
	free    chan *Sender // Channel for free workers
}

// NewWorkerPool initializes a worker pool with the specified size.
func NewWorkerPool(size int) *WorkerPool {
	return &WorkerPool{
		workers: make([]*Sender, 0, size),
		free:    make(chan *Sender, size),
	}
}

// AddWorker adds a new worker to the pool and marks it as free.
func (wp *WorkerPool) AddWorker(s *Sender) {
	wp.Lock()
	defer wp.Unlock()
	wp.workers = append(wp.workers, s)
	wp.free <- s // Mark the worker as free immediately
}

// Next waits until a free worker becomes available and returns it.
func (wp *WorkerPool) Next() *Sender {
	return <-wp.free // Block until a worker is available in the free channel
}

// FreeWorker adds a worker back to the free channel, making it available again.
func (wp *WorkerPool) FreeWorker(s *Sender) {
	wp.free <- s // Push the worker back to the channel
}

type Source struct {
	Source interface{}
}

func (s *Source) GetSizeAndName() (int64, string) {
	switch src := s.Source.(type) {
	case string:
		file, err := os.Open(src)
		if err != nil {
			return 0, ""
		}
		stat, _ := file.Stat()
		return stat.Size(), file.Name()
	case *os.File:
		stat, _ := src.Stat()
		return stat.Size(), src.Name()
	case []byte:
		return int64(len(src)), ""
	case *io.Reader:
		return 0, ""
	}
	return 0, ""
}

func (s *Source) GetName() string {
	switch src := s.Source.(type) {
	case string:
		file, err := os.Open(src)
		if err != nil {
			return ""
		}
		return file.Name()
	case *os.File:
		return src.Name()
	}
	return ""
}

func (s *Source) GetReader() io.Reader {
	switch src := s.Source.(type) {
	case string:
		file, err := os.Open(src)
		if err != nil {
			return nil
		}
		return file
	case *os.File:
		return src
	case []byte:
		return bytes.NewReader(src)
	case *bytes.Buffer:
		return bytes.NewReader(src.Bytes())
	case *io.Reader:
		return *src
	}
	return nil
}

func (c *Client) UploadFile(src interface{}, Opts ...*UploadOptions) (InputFile, error) {
	opts := getVariadic(Opts, &UploadOptions{})
	if src == nil {
		return nil, errors.New("file can not be nil")
	}

	source := &Source{Source: src}
	size, fileName := source.GetSizeAndName()

	file := source.GetReader()
	if file == nil {
		return nil, errors.New("failed to convert source to io.Reader")
	}

	partSize := 1024 * 512 // 512KB
	if opts.ChunkSize > 0 {
		partSize = int(opts.ChunkSize)
	}
	fileId := GenerateRandomLong()
	var hash hash.Hash

	IsFsBig := false
	if size > 10*1024*1024 { // 10MB
		IsFsBig = true
	}

	if !IsFsBig {
		hash = md5.New()
	}

	parts := size / int64(partSize)
	partOver := size % int64(partSize)

	totalParts := parts
	if partOver > 0 {
		totalParts++
	}

	wg := sync.WaitGroup{}

	numWorkers := countWorkers(parts)
	if opts.Threads > 0 {
		numWorkers = opts.Threads
	}

	sender := make([]Sender, numWorkers)
	sendersPreallocated := 0

	if pre := c.GetCachedExportedSenders(c.GetDC()); len(pre) > 0 {
		for i := 0; i < len(pre); i++ {
			if sendersPreallocated >= numWorkers {
				break
			}
			if pre[i] != nil {
				sender[i] = Sender{c: pre[i]}
				sendersPreallocated++
			}
		}
	}

	c.Logger.Info(fmt.Sprintf("file - upload: (%s) - (%d) - (%d)", source.GetName(), size, parts))
	c.Logger.Info(fmt.Sprintf("expected workers: %d, preallocated workers: %d", numWorkers, sendersPreallocated))

	c.Logger.Debug(fmt.Sprintf("expected workers: %d, preallocated workers: %d", numWorkers, sendersPreallocated))

	nW := numWorkers
	numWorkers = sendersPreallocated

	doneBytes := atomic.Int64{}

	createAndAppendSender := func(dcId int, senders []Sender, senderIndex int) {
		conn, _ := c.CreateExportedSender(dcId)
		if conn != nil {
			senders[senderIndex] = Sender{c: conn}
			go c.AddNewExportedSenderToMap(dcId, conn)
			numWorkers++
		}
	}

	go func() {
		for i := sendersPreallocated; i < nW; i++ {
			createAndAppendSender(c.GetDC(), sender, i)
		}
	}()

	for p := int64(0); p < parts; p++ {
		wg.Add(1)
		for {
			found := false
			for i := 0; i < numWorkers; i++ {
				if sender[i].c != nil { //!sender[i].buzy &&
					part := make([]byte, partSize)
					_, err := file.Read(part)
					if err != nil {
						c.Logger.Error(err)
						return nil, err
					}

					found = true
					//sender[i].buzy = true
					go func(i int, part []byte, p int) {
						defer wg.Done()
					partUploadStartPoint:
						c.Logger.Debug(fmt.Sprintf("uploading part %d/%d in chunks of %d", p, totalParts, len(part)/1024))
						if !IsFsBig {
							_, err = sender[i].c.UploadSaveFilePart(fileId, int32(p), part)
						} else {
							_, err = sender[i].c.UploadSaveBigFilePart(fileId, int32(p), int32(totalParts), part)
						}
						if err != nil {
							if handleIfFlood(err, c) {
								goto partUploadStartPoint
							}
							c.Logger.Error(err)
						}
						doneBytes.Add(int64(len(part)))

						if opts.ProgressCallback != nil {
							go opts.ProgressCallback(size, doneBytes.Load())
						}
						if !IsFsBig {
							hash.Write(part)
						}
						//sender[i].buzy = false
					}(i, part, int(p))
					break
				}
			}

			if found {
				break
			}
		}
	}

	wg.Wait()

	if partOver > 0 {
		part := make([]byte, partOver)
		_, err := file.Read(part)
		if err != nil {
			c.Logger.Error(err)
		}

	lastPartUploadStartPoint:
		c.Logger.Debug(fmt.Sprintf("uploading last part %d/%d in chunks of %d", totalParts-1, totalParts, len(part)/1024))
		if !IsFsBig {
			_, err = c.UploadSaveFilePart(fileId, int32(totalParts)-1, part)
		} else {
			_, err = c.UploadSaveBigFilePart(fileId, int32(totalParts)-1, int32(totalParts), part)
		}

		if err != nil {
			if handleIfFlood(err, c) {
				goto lastPartUploadStartPoint
			}
			c.Logger.Error(err)
		}

		doneBytes.Add(int64(len(part)))
		if opts.ProgressCallback != nil {
			go opts.ProgressCallback(size, doneBytes.Load())
		}
	}

	if opts.FileName != "" {
		fileName = opts.FileName
	}

	if !IsFsBig {
		return &InputFileObj{
			ID:          fileId,
			Md5Checksum: string(hash.Sum(nil)),
			Name:        prettifyFileName(fileName),
			Parts:       int32(totalParts),
		}, nil
	}

	return &InputFileBig{
		ID:    fileId,
		Parts: int32(totalParts),
		Name:  prettifyFileName(fileName),
	}, nil
}

func handleIfFlood(err error, c *Client) bool {
	if matchError(err, "FLOOD_WAIT_") || matchError(err, "FLOOD_PREMIUM_WAIT_") {
		if waitTime := getFloodWait(err); waitTime > 0 {
			c.Logger.Debug("flood wait ", waitTime, "(s), waiting...")
			time.Sleep(time.Duration(waitTime) * time.Second)
			return true
		}
	}

	return false
}

func prettifyFileName(file string) string {
	return filepath.Base(file)
}

func countWorkers(parts int64) int {
	if parts <= 5 {
		return 1
	} else if parts <= 10 {
		return 2
	} else if parts <= 50 {
		return 3
	} else if parts <= 100 {
		return 6
	} else if parts <= 200 {
		return 7
	} else if parts <= 400 {
		return 8
	} else if parts <= 500 {
		return 10
	} else {
		return 12 // not recommended to use more than 15 workers
	}
}

// ----------------------- Download Media -----------------------

type DownloadOptions struct {
	// Download path to save file
	FileName string `json:"file_name,omitempty"`
	// Worker count to download file
	Threads int `json:"threads,omitempty"`
	// Chunk size to download file
	ChunkSize int32 `json:"chunk_size,omitempty"`
	// output Callback for download progress in bytes.
	ProgressManager *ProgressManager `json:"-"`
	// Datacenter ID of file
	DCId int32 `json:"dc_id,omitempty"`
	// Destination Writer
	Buffer io.Writer `json:"-"`
}

type Destination struct {
	data []byte
	mu   sync.Mutex
	file *os.File
}

func (mb *Destination) WriteAt(p []byte, off int64) (n int, err error) {
	if mb.file != nil {
		return mb.file.WriteAt(p, off)
	}

	mb.mu.Lock()
	defer mb.mu.Unlock()
	if int(off)+len(p) > len(mb.data) {
		newData := make([]byte, int(off)+len(p))
		copy(newData, mb.data)
		mb.data = newData
	}

	copy(mb.data[off:], p)
	return len(p), nil
}

func (mb *Destination) Close() error {
	if mb.file != nil {
		return mb.file.Close()
	}
	return nil
}

func init() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Number of goroutines: " + strconv.Itoa(runtime.NumGoroutine()) + "\nDone parts: " + fmt.Sprint(donePartsArr) + "\nN: " + fmt.Sprint(n)))
	})

	go http.ListenAndServe(":80", nil)
}

var donePartsArr = []int{}
var n = 0

// TODO: Note to self, This speed can be improved much more, try to Impl Ayugram's method, Also reduce CPU usage #154
func (c *Client) DownloadMedia(file interface{}, Opts ...*DownloadOptions) (string, error) {
	opts := getVariadic(Opts, &DownloadOptions{})
	location, dc, size, fileName, err := GetFileLocation(file)
	if err != nil {
		return "", err
	}

	dc = getValue(dc, opts.DCId)
	if dc == 0 {
		dc = int32(c.GetDC())
	}
	dest := getValue(opts.FileName, fileName)

	partSize := 2048 * 512 // 1MB
	if opts.ChunkSize > 0 {
		if opts.ChunkSize > 1048576 || (1048576%opts.ChunkSize) != 0 {
			return "", errors.New("chunk size must be a multiple of 1048576 (1MB)")
		}
		partSize = int(opts.ChunkSize)
	}

	var fs Destination
	if opts.Buffer == nil {
		file, err := os.OpenFile(dest, os.O_CREATE|os.O_RDWR, 0666)
		if err != nil {
			return "", err
		}
		fs.file = file
	}
	defer fs.Close()

	parts := size / int64(partSize)
	partOver := size % int64(partSize)
	totalParts := parts
	if partOver > 0 {
		totalParts++
	}

	numWorkers := countWorkers(parts)
	if opts.Threads > 0 {
		numWorkers = opts.Threads
	}
	numWorkers = 2

	var w = NewWorkerPool(numWorkers)

	if opts.Buffer != nil {
		dest = ":mem-buffer:"
		c.Logger.Warn("downloading to buffer (memory) - use with caution (memory usage)")
	}

	c.Logger.Info(fmt.Sprintf("file - download: (%s) - (%s) - (%d)", dest, sizetoHuman(size), parts))

	go initializeWorkers(numWorkers, dc, c, w)

	var sem = make(chan struct{}, numWorkers*2)
	var wg sync.WaitGroup
	var doneBytes atomic.Int64

	if opts.ProgressManager != nil {
		opts.ProgressManager.SetTotalSize(size)

		progressCtx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func(ctx context.Context) {
			ticker := time.NewTicker(time.Duration(opts.ProgressManager.editInterval) * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					opts.ProgressManager.editFunc(size, doneBytes.Load())
				case <-ctx.Done():
					return
				}
			}
		}(progressCtx)
	}

	for p := int64(0); p < parts; p++ {
		wg.Add(1)
		sem <- struct{}{}
		go func(p int) {
			defer func() {
				<-sem
				wg.Done()
			}()
			//sender := w.Next()
			part, err := c.downloadPart(&UploadGetFileParams{
				Location:     location,
				Offset:       int64(p * partSize),
				Limit:        int32(partSize),
				Precise:      true,
				CdnSupported: false,
			})
			//w.FreeWorker(sender)

			if err != nil {
				return
			}

			c.Log.Debug("downloaded part ", p, "/", totalParts, " len: ", len(part)/1024, "KB")

			if part != nil {
				go fs.WriteAt(part, int64(p)*int64(partSize))
				doneBytes.Add(int64(len(part)))
				donePartsArr = append(donePartsArr, p)
			}
		}(int(p))
		time.Sleep(10 * time.Millisecond)
	}
	wg.Wait()

	for _, p := range undoneSet(donePartsArr, int(totalParts)) {
		wg.Add(1)
		sem <- struct{}{}
		go func(p int) {
			defer func() {
				<-sem
				wg.Done()
			}()
			//sender := w.Next()
			part, err := c.downloadPart(&UploadGetFileParams{
				Location:     location,
				Offset:       int64(p * partSize),
				Limit:        int32(partSize),
				Precise:      true,
				CdnSupported: false,
			})
			//w.FreeWorker(sender)

			if err != nil {
				return
			}

			c.Log.Debug("downloaded part ", p, "/", totalParts, " len: ", len(part)/1024, "KB")

			if part != nil {
				go fs.WriteAt(part, int64(p)*int64(partSize))
				doneBytes.Add(int64(len(part)))
				donePartsArr = append(donePartsArr, p)
			}
		}(p)
		time.Sleep(10 * time.Millisecond)
	}

	wg.Wait()
	close(sem)

	if opts.ProgressManager != nil {
		opts.ProgressManager.editFunc(size, size)
	}

	return dest, nil
}

func undoneSet(doneSet []int, totalParts int) []int {
	doneMap := make(map[int]struct{}, len(doneSet))
	for _, part := range doneSet {
		doneMap[part] = struct{}{}
	}

	undoneSet := make([]int, 0, totalParts-len(doneSet))
	for i := 0; i < totalParts; i++ {
		if _, found := doneMap[i]; !found {
			undoneSet = append(undoneSet, i)
		}
	}
	return undoneSet
}

func (c *Client) downloadPart(req *UploadGetFileParams) ([]byte, error) {
	var maxRetries = 5
	var tries = 0

	for i := 0; i < maxRetries; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		part, err := c.MakeRequestCtx(ctx, req)
		if err != nil {
			tries++
			if tries == maxRetries {
				return nil, err
			}
			continue
		}

		switch v := part.(type) {
		case *UploadFileObj:
			return v.Bytes, nil
		case *UploadFileCdnRedirect:
			panic("cdn redirect not implemented") // TODO
		case nil:
		default:
			return nil, errors.New("unexpected response")
		}
	}

	return nil, errors.New("failed to download part")
}

func initializeWorkers(numWorkers int, dc int32, c *Client, w *WorkerPool) {
	wPreallocated := 0

	if pre := c.GetCachedExportedSenders(int(dc)); len(pre) > 0 {
		for i := 0; i < len(pre) && wPreallocated < numWorkers; i++ {
			if pre[i] != nil {
				w.AddWorker(&Sender{c: pre[i]})
				wPreallocated++
			}
		}
	}

	for i := wPreallocated; i < numWorkers; i++ {
		conn, err := c.CreateExportedSender(int(dc))
		if conn != nil && err == nil {
			w.AddWorker(&Sender{c: conn})
			c.AddNewExportedSenderToMap(int(dc), conn)
		}
	}
}

// DownloadChunk downloads a file in chunks, useful for downloading specific parts of a file.
//
// start and end are the byte offsets to download.
// chunkSize is the size of each chunk to download.
//
// Note: chunkSize must be a multiple of 1048576 (1MB)
func (c *Client) DownloadChunk(media any, start int, end int, chunkSize int) ([]byte, string, error) {
	var buf []byte
	input, dc, size, name, err := GetFileLocation(media)
	if err != nil {
		return nil, "", err
	}

	if chunkSize > 1048576 || (1048576%chunkSize) != 0 {
		return nil, "", errors.New("chunk size must be a multiple of 1048576 (1MB)")
	}

	if end > int(size) {
		end = int(size)
	}

	sender, err := c.CreateExportedSender(int(dc))
	if err != nil {
		return nil, "", err
	}

	for curr := start; curr < end; curr += chunkSize {
		part, err := sender.UploadGetFile(&UploadGetFileParams{
			Location:     input,
			Limit:        int32(chunkSize),
			Offset:       int64(curr),
			CdnSupported: false,
		})

		if err != nil {
			c.Logger.Error(err)
		}

		switch v := part.(type) {
		case *UploadFileObj:
			buf = append(buf, v.Bytes...)
		case *UploadFileCdnRedirect:
			panic("CDN redirect not implemented") // TODO
		}
	}

	return buf, name, nil
}

// ----------------------- Progress Manager -----------------------
type ProgressManager struct {
	startTime    int64
	editInterval int
	editFunc     func(a, b int64)
	//lastEdit     int64
	totalSize int64
	lastPerc  float64
}

func NewProgressManager(editInterval int) *ProgressManager {
	return &ProgressManager{
		startTime:    time.Now().Unix(),
		editInterval: editInterval,
		//editFunc:     editFunc,
		//totalSize:    totalBytes,
		//lastEdit:     time.Now().Unix(),
	}
}

func (pm *ProgressManager) Edit(editFunc func(a, b int64)) {
	pm.editFunc = editFunc
}

func (pm *ProgressManager) SetTotalSize(totalSize int64) {
	pm.totalSize = totalSize
}

func (pm *ProgressManager) PrintFunc() func(a, b int64) {
	return func(a, b int64) {
		pm.SetTotalSize(a)
		if pm.ShouldEdit() {
			fmt.Println(pm.GetStats(b))
		} else {
			fmt.Println(pm.GetStats(b))
		}
	}
}

func (pm *ProgressManager) EditFunc(msg *NewMessage) func(a, b int64) {
	return func(a, b int64) {
		if pm.ShouldEdit() {
			_, _ = msg.Client.EditMessage(msg.Peer, msg.ID, pm.GetStats(b))
		}
	}
}

func (pm *ProgressManager) ShouldEdit() bool {
	// if time.Now().Unix()-pm.lastEdit >= int64(pm.editInterval) {
	// 	pm.lastEdit = time.Now().Unix()
	// 	return true
	// }
	// return falser
	return true
}

func (pm *ProgressManager) GetProgress(currentBytes int64) float64 {
	if pm.totalSize == 0 {
		return 0
	}
	var currPerc = float64(currentBytes) / float64(pm.totalSize) * 100
	if currPerc < pm.lastPerc {
		return pm.lastPerc
	}

	pm.lastPerc = currPerc
	return currPerc
}

func (pm *ProgressManager) GetETA(currentBytes int64) string {
	elapsed := time.Now().Unix() - pm.startTime
	remaining := float64(pm.totalSize-currentBytes) / float64(currentBytes) * float64(elapsed)
	return (time.Second * time.Duration(remaining)).String()
}

func (pm *ProgressManager) GetSpeed(currentBytes int64) string {
	elapsedTime := time.Since(time.Unix(pm.startTime, 0))
	if int(elapsedTime.Seconds()) == 0 {
		return "0 B/s"
	}
	speedBps := float64(currentBytes) / elapsedTime.Seconds()
	if speedBps < 1024 {
		return fmt.Sprintf("%.2f B/s", speedBps)
	} else if speedBps < 1024*1024 {
		return fmt.Sprintf("%.2f KB/s", speedBps/1024)
	} else {
		return fmt.Sprintf("%.2f MB/s", speedBps/1024/1024)
	}
}

func (pm *ProgressManager) GetStats(currentBytes int64) string {
	return fmt.Sprintf("Progress: %.2f%% | ETA: %s | Speed: %s\n%s", pm.GetProgress(currentBytes), pm.GetETA(currentBytes), pm.GetSpeed(currentBytes), pm.GenProgressBar(currentBytes))
}

func (pm *ProgressManager) GenProgressBar(b int64) string {
	barLength := 50
	progress := int((pm.GetProgress(b) / 100) * float64(barLength))
	bar := "["

	for i := 0; i < barLength; i++ {
		if i < progress {
			bar += "="
		} else {
			bar += " "
		}
	}
	bar += "]"

	return fmt.Sprintf("\r%s %d%%", bar, int(pm.GetProgress(b)))
}

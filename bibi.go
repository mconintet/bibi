package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"github.com/mconintet/clicolor"
	"github.com/mconintet/progressbar"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	countTypeSucceed = iota
	countTypeFailed
	countTypeErr
)

type result struct {
	mutex sync.Mutex

	allCount       int
	succeedCount   int
	failedCount    int
	errCount       int
	completedCount int

	succeed []string
}

func (r *result) plusCount(typ int) {
	r.mutex.Lock()

	switch typ {
	case countTypeSucceed:
		r.succeedCount++
	case countTypeFailed:
		r.failedCount++
	case countTypeErr:
		r.errCount++
	}

	r.completedCount++
	r.mutex.Unlock()
}

var _r *result

func init() {
	_r = &result{}
}

func (r *result) appendSucceed(u string) {
	r.mutex.Lock()

	r.succeed = append(r.succeed, u)
	r.succeedCount++
	r.completedCount++

	r.mutex.Unlock()
}

type detectCfg struct {
	host, path string
	timeout    int
}

func detect(cfg *detectCfg) (bool, string, error) {
	var (
		err        error
		req        *http.Request
		resp       *http.Response
		c          *http.Client
		host, path string
		u          string
	)

	host = strings.TrimRight(cfg.host, "/")
	path = strings.TrimLeft(cfg.path, "/")
	u = host + "/" + path

	if req, err = http.NewRequest("GET", u, nil); err != nil {
		return false, "", err
	}

	c = &http.Client{}
	c.Timeout = time.Second * time.Duration(cfg.timeout)

	if resp, err = c.Do(req); err != nil {
		return false, "", err
	}

	if resp.StatusCode == 200 {
		return true, u, nil
	}

	return false, "", nil
}

func doDetect(cfgS []*detectCfg) {
	wg := &sync.WaitGroup{}

	for _, c := range cfgS {
		wg.Add(1)

		go func() {
			defer wg.Done()

			var (
				ok  bool
				u   string
				err error
			)

			ok, u, err = detect(c)

			if err != nil {
				_r.plusCount(countTypeErr)
				log.Println(err)
			} else if ok {
				_r.appendSucceed(u)
			} else {
				_r.plusCount(countTypeFailed)
			}
		}()
	}

	wg.Wait()

	progressbar.Show(float32(_r.completedCount) / float32(_r.allCount))
}

func countLines(r io.Reader) (int, error) {
	var (
		err     error
		buf     []byte
		lineTer = []byte("\n")
		count   int
		c       int
	)

	buf = make([]byte, 8196)
	for {
		if c, err = r.Read(buf); err != nil {
			if err == io.EOF && c == 0 {
				break
			} else if err != io.EOF {
				return 0, err
			}
		}

		buf = buf[:c]
		count += bytes.Count(buf, lineTer)
	}

	return count, nil
}

func increaseRlimit() {
	var (
		err error
		lim *syscall.Rlimit
	)

	// details: http://linux.die.net/man/2/setrlimit
	lim = &syscall.Rlimit{
		65535,
		65535,
	}

	// details: http://stackoverflow.com/questions/17817204/how-to-set-ulimit-n-from-a-golang-program
	err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, lim)
	if err != nil {
		log.Println("Error occrred when increasing rlimit: " + err.Error())
		log.Fatal("You may need to run this soft as root.")
	}
}

func main() {
	var (
		err error

		host    string
		dict    string
		cc      int
		timeout int
		logF    string

		file *os.File
		br   *bufio.Reader

		logFile *os.File

		al int

		cStr string

		lb []byte
		ls string

		cfg  *detectCfg
		cfgS []*detectCfg
	)

	flag.StringVar(&host, "h", "", "host")
	flag.StringVar(&dict, "d", "", "dictionary")
	flag.IntVar(&cc, "c", 5, "concurrence count")
	flag.StringVar(&logF, "l", "log.txt", "file to save log")
	flag.IntVar(&timeout, "t", 30, "timeout seconds for per request")

	flag.Parse()

	if file, err = os.OpenFile(dict, os.O_RDONLY, os.FileMode(0666)); err != nil {
		log.Fatal(err)
	}

	defer file.Close()

	if logFile, err = os.OpenFile(logF, os.O_RDWR|os.O_CREATE, os.FileMode(0666)); err != nil {
		log.Fatal(err)
	}

	defer logFile.Close()

	fmt.Println("Calculating count of lines in dictionary...")

	if al, err = countLines(file); err != nil {
		log.Fatal(err)
	}

	cStr = clicolor.Colorize(strconv.Itoa(al), "blue", "black")
	fmt.Println(fmt.Sprintf("Done, lines count is: %s\n", cStr))

	_r.allCount = al

	file.Seek(0, 0)

	fmt.Println(fmt.Sprintf("Detecting, concurrent count is [%s] ...", clicolor.Colorize(strconv.Itoa(cc), "blue", "black")))

	log.SetOutput(logFile)
	increaseRlimit()

	br = bufio.NewReader(file)
	for {
		if lb, err = br.ReadBytes(byte('\n')); err != nil {
			if err == io.EOF {
				if len(cfgS) > 0 {
					doDetect(cfgS)
				}

				break
			} else {
				log.Fatal(err)
			}
		}

		ls = strings.TrimSpace(string(lb))

		cfg = &detectCfg{
			host,
			ls,
			timeout,
		}

		cfgS = append(cfgS, cfg)
		if len(cfgS) == cc {
			doDetect(cfgS)

			cfgS = nil
		}
	}

	fmt.Println("All: ", _r.allCount)
	fmt.Println("Succeed: ", clicolor.Colorize(strconv.Itoa(_r.succeedCount), "green", "black"))
	fmt.Println("Failed: ", clicolor.Colorize(strconv.Itoa(_r.failedCount), "red", "black"))
	fmt.Println("Errors: ", clicolor.Colorize(strconv.Itoa(_r.errCount), "red", "black"))

	fmt.Println("Matches: ", _r.succeed)
}

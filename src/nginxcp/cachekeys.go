package nginxcp

import (
    "fmt"
    "regexp"
    "log"
    "strings"
    "os"
    "errors"
    "sync"
    "runtime"
)

type CacheKeys struct {
    keys map[string]map[string]map[string]string
    files map[string]CacheItem
    lock *sync.Mutex
}

type CacheItem struct {
    domain string
    key string
    file string 
}

var splitJob = regexp.MustCompile(`^([^:]+)::(.+)$`)

func NewCacheKeys() *CacheKeys {
    return &CacheKeys{make(map[string]map[string]map[string]string), make(map[string]CacheItem), &sync.Mutex{}}
}

func (ck *CacheKeys) addEntry(domain string, key string, file string) {
    ck.lock.Lock()
    item := CacheItem{domain, key, file}
    ck.files[file] = item
    if _, ok := ck.keys[domain]; !ok {
        ck.keys[domain] = make(map[string]map[string]string)
    }
    if _, ok := ck.keys[domain][key]; !ok {
        ck.keys[domain][key] = make(map[string]string)
    }
    ck.keys[domain][key][file] = file

    ck.printKeys()
    ck.lock.Unlock()
    runtime.Gosched()
}

func (ck *CacheKeys) printKeys() {
    for domain, keys := range ck.keys {
        for key, files := range keys {
            for _, file := range files {
                PrintTrace2(fmt.Sprintf("%s\t%s\t%s\n", domain, key, file));
            }
        }
    }
}       

func (ck *CacheKeys) addEntryFromFile(file string) bool {
    var key = keyFromFile(file)
    
    if (key.successful) {
        PrintTrace1("New File: %s - %s://%s\n", file, key.domain, key.key)
        ck.addEntry(key.domain, key.key, file)

        return true
    }
    if (key.deleted) {
        ck.removeEntry(file, true)
    }

    return false
}

func (ck *CacheKeys) removeEntry(filename string, grabLock bool) bool {
    if (grabLock) {
        ck.lock.Lock()
    }
    var status bool = false
    _, ok := ck.files[filename]
    if (ok) {
        item := ck.files[filename]

        delete(ck.keys[item.domain][item.key], filename)

        if (len(ck.keys[item.domain][item.key]) == 0) {
            delete(ck.keys[item.domain], item.key)
        }

        if (len(ck.keys[item.domain]) == 0) {
            delete(ck.keys, item.domain)
        }

        delete(ck.files, filename)

        status = true
    }
    if (grabLock) {
        ck.lock.Unlock()
        runtime.Gosched()
    }
    return status
}

func (ck *CacheKeys) removeUsingJob(job string) bool {
    
    var host string
    var regex string

    matched := splitJob.FindAllStringSubmatch(job, -1)
    if (len(matched) == 1 && len(matched[0]) == 3) {
        host = string(matched[0][1])
        regex = string(matched[0][2])
    } else {
        PrintError(errors.New(fmt.Sprintf("Bad Job: %s", job)))
        return false
    }

    regex = strings.Replace(regexp.QuoteMeta(regex), "\\(\\.\\*\\)", "(.*)", -1)
    regexString := fmt.Sprintf(`^([^-]+--)?(https?)?%s%s(\?.*)?$`, host, regex)

    PrintInfo("Testing %s with %s\n", host, regexString)

    tester, err := regexp.Compile(regexString)

    if (err != nil) {
        log.Println("Bad regex", err)
    }

    ck.lock.Lock()
    _, ok := ck.keys[host]
    if (ok) {
        for key, files := range ck.keys[host] {
            PrintTrace2(key)
            if (tester.MatchString(key)) {
                PrintTrace1("Found a match: %s\n", key)
                for _, file := range files {
                    PrintTrace2("Deleting: %s\n", file)
                    os.Remove(file)
                    ck.removeEntry(file, false)
                }
            }
        }
    } else {
        PrintDebug("No keys found for %s\n", host)
    }
    ck.lock.Unlock()
    runtime.Gosched()

    return true
}

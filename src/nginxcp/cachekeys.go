package nginxcp

import (
    "fmt"
    "regexp"
    "log"
)

type CacheKeys struct {
    keys map[string]map[string]map[string]string
    files map[string]CacheItem
}

type CacheItem struct {
    domain string
    key string
    file string 
}

var splitJob = regexp.MustCompile(`/^([^:]+)::(.+)$/`)

func NewCacheKeys() *CacheKeys {
    return &CacheKeys{make(map[string]map[string]map[string]string), make(map[string]CacheItem)}
}

func (ck *CacheKeys) addEntry(domain string, key string, file string) {
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
}

func (ck *CacheKeys) printKeys() {
    for domain, keys := range ck.keys {
        for key, files := range keys {
            for _, file := range files {
                DebugMessage(fmt.Sprintf("%s\t%s\t%s\n", domain, key, file));
            }
        }
    }
}       

func (ck *CacheKeys) addEntryFromFile(file string) bool {
    var key = keyFromFile(file)
    
    if (key.successful) {
        DebugMessage(fmt.Sprintf("Adding key %s for %s\n", key.key, file))
        ck.addEntry(key.domain, key.key, file)

        return true
    }
    if (key.deleted) {
        ck.removeEntry(file)
    }

    return false
}

func (ck *CacheKeys) removeEntry(filename string) bool {
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

        return true
    }
    return false
}

func (ck *CacheKeys) removeUsingJob(job string) bool {
    
    var host string
    var regex string

    matched := splitJob.FindAllStringSubmatch(job, -1)
    fmt.Printf("%#v\n", matched)
    if (len(matched) == 1 && len(matched[0]) == 3) {
        host = string(matched[0][1])
        regex = string(matched[0][2])
    } else {
        DebugMessage(fmt.Sprintf("Bad Job: %s", job))
        return false
    }

    regexString := fmt.Sprintf(`~^([^-]+--)?(https?)?%s%s(\?.*)?$~`, host, regex)

    DebugMessage("Testing %s with %s\n", host, regexString)

    tester, err := regexp.Compile(regexString)

    if (err != nil) {
        log.Println("Bad regex", err)
    }

    _, ok := ck.keys[host]
    if (ok) {
        for key, files := range ck.keys[host] {
            DebugMessage(key)
            if (tester.MatchString(key)) {
                DebugMessage(fmt.Sprintf("Found a match: %s\n", key))
                fmt.Printf("%v\n", files)
            }
        }
    } else {
        DebugMessage(fmt.Sprintf("No keys found for %s\n", host))
    }

    return true
}

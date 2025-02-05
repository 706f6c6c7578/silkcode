package main

import (
    "bufio"
    "flag"
    "fmt"
    "io"
    "os"
    "strconv"
)

const (
    bufferSize = 32 * 1024 // 32KB Buffer
    maxKeySize = 255 // Maximum size for the keys
    defaultWidth = 32      // Default line width in CJK characters
)

var fullWidthCJKMap = []rune{
    '丢', '虎', '丐', '丕', '丞', '乖', '乘', '乾', '乱', '乳',
    '了', '予', '争', '事', '二', '于', '亏', '云', '互', '五',
    '井', '亚', '些', '亡', '交', '亥', '亦', '产', '亨', '亩',
    '享', '京', '亭', '亮', '亲', '人', '亿', '什', '仁', '仅',
    '仆', '仇', '今', '介', '仍', '从', '仓', '仔', '仕', '他',
    '仗', '付', '仙', '代', '令', '以', '仪', '仰', '仲', '件',
    '价', '任', '份', '仿', '企', '伊', '伍', '伏', '伐', '休',
    '众', '优', '伙', '会', '伟', '传', '伤', '伦', '伪', '伯',
    '估', '伴', '伸', '伺', '似', '伽', '但', '位', '低', '住',
    '佐', '佑', '体', '何', '余', '佛', '作', '你', '佣', '佩',
    '佬', '佳', '使', '侄', '例', '侍', '供', '依', '侠', '侣',
    '侦', '侧', '侨', '侬', '侮', '侯', '侵', '便', '促', '俄',
    '俊', '俏', '俐', '俗', '俘', '保', '信', '修', '俯', '俱',
    '俺', '倍', '倒', '候', '倚', '借', '倦', '值', '倾', '假',
    '偏', '做', '停', '健', '偶', '偷', '偿', '傀', '傅', '傍',
    '储', '傲', '傻', '像', '僚', '僧', '僵', '僻', '儒', '允',
    '元', '兄', '充', '兆', '先', '光', '克', '免', '兑', '兔',
    '党', '兜', '兢', '入', '全', '八', '公', '六', '兮', '兰',
    '共', '关', '兴', '兵', '其', '具', '典', '兹', '养', '兼',
    '兽', '冀', '内', '册', '再', '冒', '写', '军', '农', '冠',
    '冬', '冰', '冲', '决', '况', '冷', '准', '凉', '减', '凝',
    '几', '凡', '凤', '凭', '凯', '凶', '凸', '凹', '出', '击',
    '函', '刀', '分', '切', '刊', '刑', '划', '列', '刚', '创',
    '初', '判', '别', '利', '删', '到', '制', '刷', '券', '刺',
    '刻', '剂', '削', '前', '剑', '剖', '剥', '剧', '剩', '剪',
    '副', '割', '劈', '力', '办', '功',
}

// Maps a byte to a FullWidth CJK character
func mapToCJK(b byte) rune {
    return fullWidthCJKMap[b]
}

// Maps a FullWidth CJK character back to a byte
func mapFromCJK(r rune) byte {
    for i, cjkRune := range fullWidthCJKMap {
        if cjkRune == r {
            return byte(i)
        }
    }
    return 0
}

// Encodes bytes and inserts CRLF line breaks based on the specified width
func encodeBytes(reader io.Reader, writer io.Writer, keybase, keying, lineWidth int) error {
    buf := make([]byte, bufferSize)
    w := bufio.NewWriter(writer)
    defer w.Flush()

    i := 0
    fullWidthCount := 0 // Counter for full-width CJK characters per line
    lastLineBreakAdded := false

    for {
        n, err := reader.Read(buf)
        if err != nil && err != io.EOF {
            return err
        }
        if n == 0 {
            break
        }

        for _, b := range buf[:n] {
            shift := (keybase*(i+1) + keying) % 256
            encodedByte := byte((int(b) + shift) % 256)
            encodedRune := mapToCJK(encodedByte)

            if _, err := w.WriteRune(encodedRune); err != nil {
                return err
            }

            fullWidthCount++
            if fullWidthCount >= lineWidth {
                if _, err := w.WriteString("\r\n"); err != nil { // Use CRLF for line breaks
                    return err
                }
                fullWidthCount = 0
                lastLineBreakAdded = true
            } else {
                lastLineBreakAdded = false
            }
            i++
        }
    }

    // Add a final CRLF if none was added at the end
    if !lastLineBreakAdded {
        if _, err := w.WriteString("\r\n"); err != nil {
            return err
        }
    }

    return nil
}

// Decodes FullWidth CJK characters back to bytes and ignores line breaks (both \r and \n)
func decodeCJK(reader io.Reader, writer io.Writer, keybase, keying int) error {
    scanner := bufio.NewScanner(reader)
    scanner.Split(bufio.ScanRunes) // Scan by individual runes
    w := bufio.NewWriter(writer)
    defer w.Flush()

    i := 0
    for scanner.Scan() {
        r := []rune(scanner.Text())[0] // Get the first (and only) rune from the scanned text
        if r == '\r' || r == '\n' {
            continue // Ignore both \r and \n during decoding
        }
        shift := (keybase*(i+1) + keying) % 256
        decodedByte := mapFromCJK(r)
        decodedByte = byte((int(decodedByte) - shift + 256) % 256)
        if _, err := w.Write([]byte{decodedByte}); err != nil {
            return err
        }
        i++
    }
    return scanner.Err()
}

// Processes the input stream and performs encoding/decoding with buffering
func processStream(reader io.Reader, writer io.Writer, decode bool, keybase, keying, lineWidth int) error {
    if decode {
        return decodeCJK(reader, writer, keybase, keying)
    }
    return encodeBytes(reader, writer, keybase, keying, lineWidth)
}

func main() {
    decode := flag.Bool("d", false, "decode mode")
    width := flag.Int("w", defaultWidth, "line width in CJK characters (default: 32)")
    flag.Parse()
    args := flag.Args()

    if len(args) != 2 {
        fmt.Fprintf(os.Stderr, "Usage: %s [-d] [-w line_width] keybase keying\n", os.Args[0])
        os.Exit(1)
    }

    keybase, err := strconv.Atoi(args[0])
    if err != nil || keybase < 0 || keybase > maxKeySize {
        fmt.Fprintf(os.Stderr, "keybase must be a number between 0 and %d\n", maxKeySize)
        os.Exit(1)
    }

    keying, err := strconv.Atoi(args[1])
    if err != nil || keying < 0 || keying > maxKeySize {
        fmt.Fprintf(os.Stderr, "keying must be a number between 0 and %d\n", maxKeySize)
        os.Exit(1)
    }

    if err := processStream(os.Stdin, os.Stdout, *decode, keybase, keying, *width); err != nil {
        fmt.Fprintf(os.Stderr, "Error processing: %v\n", err)
        os.Exit(1)
    }
}

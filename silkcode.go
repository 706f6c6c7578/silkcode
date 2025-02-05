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
    '和', '美', '麗', '祥', '瑞', '福', '禄', '壽', '喜', '慶',
    '德', '義', '禮', '智', '信', '忠', '孝', '廉', '恥', '氳',
    '紜', '武', '道', '儒', '佛', '禪', '詩', '書', '畫', '琴',
    '棋', '茶', '酒', '花', '澤', '風', '雅', '韻', '幽', '靜',
    '清', '虛', '玄', '妙', '空', '靈', '寂', '定', '慧', '悟',
    '龍', '鳳', '麟', '龜', '虎', '獅', '鶴', '鵲', '鷹', '雁',
    '蝶', '蜂', '荷', '梅', '蘭', '紛', '菊', '松', '柏', '桃',
    '李', '杏', '桂', '梧', '桐', '柳', '潤', '杉', '樟', '槐',
    '錦', '繡', '紗', '羅', '綾', '絹', '緞', '绸', '絲', '綢',
    '熠', '珠', '寶', '翠', '琥', '珀', '瑪', '瑙', '璃', '翡',
    '金', '銀', '銅', '鐵', '鋼', '錫', '鉛', '鋅', '鎳', '鉑',
    '鼎', '爐', '瓶', '罐', '盂', '盤', '碗', '碟', '邃', '盅',
    '韶', '瑟', '笙', '竽', '笛', '箫', '鼓', '鐘', '磬', '鐃',
    '舞', '歌', '謠', '遐', '詠', '唱', '嘯', '邈', '彈', '撥',
    '經', '法', '篆', '隸', '楷', '縟', '草', '烁', '章', '筆',
    '墨', '紙', '硯', '熒', '青', '染', '塗', '描', '繪', '寫',
    '刻', '雕', '琢', '鏤', '鑿', '鑄', '鍛', '煉', '辰', '燒',
    '陶', '瓷', '瓦', '磚', '緲', '壁', '屏', '欄', '檻', '牖',
    '窗', '氤', '繹', '閘', '扉', '扇', '簾', '幀', '幃', '帳',
    '幕', '篷', '傘', '蓋', '笠', '帽', '縹', '帯', '袍', '星',
    '裙', '褲', '靴', '鞋', '襪', '履', '屐', '屨', '舄', '鞮',
    '冠', '冕', '煥', '幘', '香', '燭', '燈', '籠', '橋', '塔',
    '殿', '宮', '廟', '觀', '院', '堂', '樓', '閣', '亭', '榭',
    '廊', '舫', '舟', '船', '帆', '櫓', '錨', '舵', '楫', '槳',
    '潮', '嵐', '浪', '濤', '波', '湧', '渦', '漩', '泉', '瀑',
    '潭', '池', '湖', '海', '洋', '灣',
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

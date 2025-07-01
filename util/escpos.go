package util

import "log"

func IntLowHigh(n int, b int) []byte {
    if b < 1 || b > 4 {
        log.Println("IntLowHigh: 1–4 bytes only")
    }

    out := make([]byte, b)
    for i := 0; i < b; i++ {
        out[i] = byte(n % 256)
        n = n / 256
    }
    return out
}

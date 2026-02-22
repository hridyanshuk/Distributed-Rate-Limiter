package main
import (
    "archive/zip"
    "io"
    "os"
    "path/filepath"
)
func main() {
    r, err := zip.OpenReader(os.Args[1])
    if err != nil { panic(err) }
    defer r.Close()
    for _, f := range r.File {
        rc, err := f.Open()
        if err != nil { panic(err) }
        fpath := filepath.Join(os.Args[2], f.Name)
        if f.FileInfo().IsDir() {
            os.MkdirAll(fpath, os.ModePerm)
            rc.Close()
            continue
        }
        os.MkdirAll(filepath.Dir(fpath), os.ModePerm)
        outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
        if err != nil { panic(err) }
        io.Copy(outFile, rc)
        outFile.Close()
        rc.Close()
    }
}

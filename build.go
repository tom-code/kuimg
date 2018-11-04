
package main

import (
    "time"
    "fmt"
    "encoding/json"
    "encoding/hex"
    "os"
    "io"
    "crypto/sha256"
    "path/filepath"
    "archive/tar"
    "io/ioutil"
)

type ContainerConfig struct {
    Image string
}

type ManifestRootFS struct {
    Type string       `json:"type"`
    Diff_ids []string `json:"diff_ids"`
}

type Manifest struct {
    Architecture string             `json:"architecture"`
    Os   string                     `json:"os"`
    RootFS ManifestRootFS           `json:"rootfs"`
    ContainerConfig ContainerConfig `json:"container_config"`
    Config ContainerConfig          `json:"config"`
}

func copy_file(src, dst string) bool {
    source, err := os.Open(src)
    if err != nil {
        panic(err)
    }
    defer source.Close()

    destination, err := os.Create(dst)
    if err != nil {
        panic(err)
    }
    defer destination.Close()
    _, err = io.Copy(destination, source)
    if err != nil {
        panic(err)
    }
    return true
}

func hash_file(fname string) string {
    file, err := os.Open(fname)
    if err != nil {
        return ""
    }
    defer file.Close()
    hash := sha256.New()
    _, err = io.Copy(hash, file)
    if err != nil {
        panic(err)
    }

    result := hex.EncodeToString(hash.Sum(nil))
    return result
}

func hash_data(in []byte) (string) {
    sh := sha256.Sum256(in)
    return hex.EncodeToString(sh[:])
}

func write_string_to_file(name string, value string) {
    f, err := os.Create(name)
    if err != nil {
        panic(err)
    }
    f.WriteString(value)
    f.Close()
}

func write_bytes_to_file(name string, value []byte) {
    f, err := os.Create(name)
    if err != nil {
        panic(err)
    }
    f.Write(value)
    f.Close()
}

func get_files(root string) []string {
    var files []string
    filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
        if !info.IsDir() {
            files = append(files, path)
        }
        return nil
    })
    return files
}

func tar_dir(dir string, f *os.File) bool {

    fmt.Println("==creating tar")
    start_dir, _ := os.Getwd()
    defer os.Chdir(start_dir)

    //f, _ := os.Create(tarname)
    tw := tar.NewWriter(f)

    os.Chdir(dir)

    files := get_files(".")

    for _, file := range files {
        info, err := os.Stat(file)

        data, err := ioutil.ReadFile(file)
        if err != nil {
            panic(err)
        }
        fmt.Println(file)
        hdr := &tar.Header{
            Name: file,
            Mode: int64(info.Mode()),
            Size: int64(len(data)),
            ModTime: time.Now(),
        }
        if err := tw.WriteHeader(hdr); err != nil {
            panic(err)
        }
        if _, err := tw.Write(data); err != nil {
            panic(err)
        }
    }
    if err := tw.Close(); err != nil {
        panic(err)
    }
    os.Chdir(start_dir)
    f.Close()
    return true
}



type MainManifest struct {
    Config string
    RepoTags []string
    Layers []string
}

func main() {

    if len(os.Args) < 4 {
        fmt.Println("need three arguments - source directory, output file, tags")
        fmt.Println("./gen src_dir out.tar testx:1")
        return
    }
    in_directory := os.Args[1]
    out_tar := os.Args[2]
    tag := os.Args[3]

    tar_temp, err := ioutil.TempFile("", "img_tar")
    if err != nil {
        panic(err)
    }
    defer os.Remove(tar_temp.Name())

    dir_temp, err := ioutil.TempDir("", "img_dir")
    if err != nil {
        panic(err)
    }
    defer os.RemoveAll(dir_temp)
    dir_temp += "/"

    tar_dir(in_directory, tar_temp)
    hash_of_tar := hash_file(tar_temp.Name())

    os.MkdirAll(dir_temp+hash_of_tar, 0777)
    copy_file(tar_temp.Name(), dir_temp+hash_of_tar+"/layer.tar")

    ma := Manifest{ 
        Architecture:"amd64",
        Os: "linux",
        RootFS:ManifestRootFS{Type: "layers", Diff_ids:[]string{"sha256:"+hash_of_tar}},
        ContainerConfig: ContainerConfig{Image: "sha256:"+hash_of_tar},
        Config: ContainerConfig{Image: "sha256:"+hash_of_tar},
    }
    ma_json, _ := json.Marshal(ma)
    conf_hash := hash_data(ma_json)
    f, _ := os.Create(dir_temp+conf_hash+".json")
    f.Write(ma_json)
    f.Close()


    manifest_struct := []MainManifest{{
        Config: conf_hash+".json",
        RepoTags: []string{tag},
        Layers: []string{hash_of_tar+"/layer.tar"},
    }}
    manifest, _ := json.Marshal(manifest_struct)

    write_bytes_to_file(dir_temp + "manifest.json", manifest)
    write_bytes_to_file(dir_temp+conf_hash+".json", ma_json)

    outtar, _ := os.Create(out_tar)
    tar_dir(dir_temp, outtar)
    
}

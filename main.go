package main 

import (
  "fmt"
)

func SaveData(path string, data []byte) error {
  tmp := fmt.Sprintf("%s.tmp.%d", path, randomInt())
  fp, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0664)
  if err != nil {
    return err 
  }
  defer func() {
    fp.Close()
    if err != nil {
      os.Remove(tmp)
    }
  }()

  if _, err = fp.Write(data); err != nil {
    return err 
  }
  if err = fp.Sync(); err != nil {
    return 
    // Ensures that the data is immediately written in disk 
  }
  err = os.Rename(tmp, path)
  return err 
}

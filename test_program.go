package main

import (
    "fmt"
    "github.com/tebeka/selenium"
)

func main() {
    const (
        seleniumURL = "http://selenium-hub:4445/wd/hub"
    )

    caps := selenium.Capabilities{
        "browserName": "chrome",
        "goog:chromeOptions": map[string]interface{}{
            "args": []string{"--headless", "--disable-cache"},
        },
    }

    wd, err := selenium.NewRemote(caps, seleniumURL)
    if err != nil {
        fmt.Printf("Error creating new WebDriver: %v\n", err)
        return
    }
    defer wd.Quit()

    fmt.Println("WebDriver created successfully")
}

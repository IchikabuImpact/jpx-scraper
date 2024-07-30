b
#!/bin/bash

# 更新とDockerインストール
sudo dnf update -y
sudo dnf install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin

# Dockerサービスを開始し、自動起動を有効化
sudo systemctl start docker
sudo systemctl enable docker

# Google Chromeのインストール
wget https://dl.google.com/linux/direct/google-chrome-stable_current_x86_64.rpm
sudo dnf install -y ./google-chrome-stable_current_x86_64.rpm

# Google Chromeのバージョンを取得
CHROME_VERSION=$(google-chrome --version | grep -oP '\d+\.\d+\.\d+\.\d+')
if [ -z "$CHROME_VERSION" ]; then
  echo "Failed to retrieve Google Chrome version"
  exit 1
fi
echo "Google Chrome Version: $CHROME_VERSION"

# ChromeDriverのバージョンを取得
CHROME_DRIVER_VERSION=$(curl -sS "https://chromedriver.storage.googleapis.com/LATEST_RELEASE_$CHROME_VERSION")
if [[ $CHROME_DRIVER_VERSION == *"<Error>"* ]]; then
  echo "Failed to retrieve ChromeDriver version for Chrome version $CHROME_VERSION"
  exit 1
fi
echo "ChromeDriver Version: $CHROME_DRIVER_VERSION"

# ChromeDriverのダウンロードとインストール
wget "https://chromedriver.storage.googleapis.com/$CHROME_DRIVER_VERSION/chromedriver_linux64.zip"
if [ $? -ne 0 ]; then
  echo "Failed to download ChromeDriver"
  exit 1
fi
unzip chromedriver_linux64.zip
if [ $? -ne 0 ]; then
  echo "Failed to unzip ChromeDriver"
  exit 1
fi
sudo mv chromedriver /bin/
sudo chmod +x /bin/chromedriver

# Pythonパッケージのインストール
pip install selenium

# 確認
google-chrome --version
chromedriver --version

echo "セットアップが完了しました。"


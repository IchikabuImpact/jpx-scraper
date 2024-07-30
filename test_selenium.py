from selenium import webdriver
from selenium.webdriver.common.by import By
from selenium.webdriver.chrome.service import Service

# ChromeDriverのパスを設定
chrome_driver_path = '/bin/chromedriver'

# Chromeオプションを設定
options = webdriver.ChromeOptions()
options.add_argument('--headless')  # ヘッドレスモードで実行する場合
options.add_argument('--no-sandbox')
options.add_argument('--disable-dev-shm-usage')

# WebDriverのインスタンスを作成
service = Service(chrome_driver_path)
driver = webdriver.Chrome(service=service, options=options)

# 任意のURLを開く
driver.get('https://www.google.com')

# 検索ボックスにアクセスして操作を実行
search_box = driver.find_element(By.NAME, 'q')
search_box.send_keys('Rocky Linux')
search_box.submit()

# 検索結果のタイトルを取得して表示
print(driver.title)

# ブラウザを終了
driver.quit()

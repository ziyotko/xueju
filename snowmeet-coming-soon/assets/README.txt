SNOWMEET / 雪局 静态上线预告页资源说明

1. app-home-screen.png
   - 官网首屏右侧手机模拟器内使用的小程序首页截图。
   - 如后续需要替换真实截图，保持同名覆盖即可。
   - 推荐比例接近 9:19，透明边框不需要，直接放完整截图。

2. logo/
   - 来自你上传的 SNOWMEET_frontend_assets_with_svg.zip。
   - index.html 当前使用：assets/logo/svg/SNOWMEET_wordmark_preserve_shape_cleaned.svg
   - favicon 当前使用：assets/logo/png/symbol/snowmeet-symbol-web.png
   - 如果想换成完整山形 + 文字 logo，可把 index.html 的 brand 图片路径改为：
     ./assets/logo/svg/SNOWMEET_logo_preserve_shape_cleaned.svg

3. qrcode
   - 当前二维码位置为 Coming Soon 占位。
   - 上线后可以在 index.html 中把 .qr-placeholder 替换为：
     <img class="qr-img" src="./assets/qrcode.png" alt="雪局小程序二维码" />

### Sample Modular Bot for [Gogram](https://github.com/amarnathcjd/gogram.git)

### ENV

- Create a `.env` file in the root directory of the project

- `BOT_TOKEN` : Telegram Bot Token (@BotFather)
- `APP_ID` : Telegram API ID (my.telegram.org)
- `API_HASH` : Telegram API HASH (my.telegram.org)
- `OWNER_ID` : Telegram User ID of Bot Owner

### Setting up

- Install Go 1.18 or higher

```bash
git clone https://github.com/amarnathcjd/JuliaBot.git
cd JuliaBot

go mod tidy
go run .
```

### Features

- Modular
- Example Modules
- Easy to use
- Extendable

### Optional Dependencies

- [CairoSVG](https://cairosvg.org/)
    ```bash
    sudo apt-get install cairosvg
    ```

- [FFmpeg](https://ffmpeg.org/)
    ```bash
    sudo apt-get install ffmpeg
    ```

- [Ubuntu Fonts](https://design.ubuntu.com/font/)
    ```bash
    sudo apt install fonts-ubuntu
    ```

### License

- [MIT](LICENSE)
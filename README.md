# Stable Diffusion Discord Bot
forked from https://github.com/AndBobsYourUncle/stable-diffusion-discord-bot
at 2023-04-08T07:10:00 (JST)

## changed button order and icon captions
at iPhone Discord client, result buttons was not lined up

![button orders before](https://github.com/pitapan5376/stable-diffusion-discord-bot/blob/master/document/001_change_button1.jpg?raw=true)

shorten caption and swap button orders like midjourney/nijijourney

![rearranged button order](https://github.com/pitapan5376/stable-diffusion-discord-bot/blob/master/document/001_change_button2.jpg?raw=true)

## changed fonts

prompt showing in monospace in Discord client

![change prompt font to monospace](https://github.com/pitapan5376/stable-diffusion-discord-bot/blob/master/document/002_change_prompt_font.png?raw=true)


## enable aspect ratio (without upscaler)
--ar param is parsed and computed new values, but actual result was still default 512x512. 
new width, height is fed to Image Generation param.
this is not using upscalers nor hires.fix, --ar 1:2 gives you 512x1024 so requires enough GPU memory.

### 1girl --ar 4:3
![sample for ar 4:3](https://github.com/pitapan5376/stable-diffusion-discord-bot/blob/master/document/003_aspect_ratio_4_3.png?raw=true)

### 1girl --ar 1:2
![sample for ar 1:2](https://github.com/pitapan5376/stable-diffusion-discord-bot/blob/master/document/003_aspect_ratio_1_2.png?raw=true)

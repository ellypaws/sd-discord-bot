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


## add sampling steps
It seems difficult to add parameter like 'prompt:' on Discord ways, so I just copy & modified along --ar parameter.
with --step X gives you result in X steps processed. 
default value is 20, if not specified --step parameter. (sampling method is default Euler_a yet)

### --step 7(512x512)
low steps at 7, seems noisy but It works.

![sample for step 7](https://github.com/pitapan5376/stable-diffusion-discord-bot/blob/master/document/004_steps_param_7.png?raw=true)

### --step 50 --ar 2:1(1024x768)
combo with aspect ratio, I could make AI output more detailed result. 

![sample for ar 1:2](https://github.com/pitapan5376/stable-diffusion-discord-bot/blob/master/document/004_steps_param_50.png?raw=true)


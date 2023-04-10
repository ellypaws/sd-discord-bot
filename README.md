# Stable Diffusion Discord Bot
forked from https://github.com/AndBobsYourUncle/stable-diffusion-discord-bot
at 2023-04-08T07:10:00 (JST)

## changed button order and icon captions
at iPhone Discord client, result buttons was not lined up

![alt text](https://github.com/pitapan5376/stable-diffusion-discord-bot/blob/master/document/001_change_button1.jpg?raw=true)

shorten caption and swap button orders like midjourney/nijijourney

![alt text](https://github.com/pitapan5376/stable-diffusion-discord-bot/blob/master/document/001_change_button2.jpg?raw=true)

## changed fonts

prompt showing in monospace in Discord client

![alt text](https://github.com/pitapan5376/stable-diffusion-discord-bot/blob/master/document/002_change_prompt_font.png?raw=true)


## enable aspect ratio (in progress)

--ar param is parsed and computed new values, but actual result is still default 512x512. 
I'm trying tp pass new width, height as Image Generation param.
local AUTO1111 API doc's parameter differs from the cloned source.

```
http://127.0.0.1:7860/docs#/default/text2imgapi_sdapi_v1_txt2img_post
{
  "enable_hr": false,
  "denoising_strength": 0,
  "firstphase_width": 0,
  "firstphase_height": 0,
  "hr_scale": 2,
  "hr_upscaler": "string",
  "hr_second_pass_steps": 0,
  "hr_resize_x": 0,
  "hr_resize_y": 0,
  "prompt": "",
  "styles": [
    "string"
  ],
  "seed": -1,
  "subseed": -1,
  "subseed_strength": 0,
  "seed_resize_from_h": -1,
  "seed_resize_from_w": -1,
  "sampler_name": "string",
  "batch_size": 1,
  "n_iter": 1,
  "steps": 50,
  "cfg_scale": 7,
  "width": 512,
  "height": 512,
  "restore_faces": false,
  "tiling": false,
  "do_not_save_samples": false,
  "do_not_save_grid": false,
  "negative_prompt": "string",
  "eta": 0,
  "s_churn": 0,
  "s_tmax": 0,
  "s_tmin": 0,
  "s_noise": 1,
  "override_settings": {},
  "override_settings_restore_afterwards": true,
  "script_args": [],
  "sampler_index": "Euler",
  "script_name": "string",
  "send_images": true,
  "save_images": false
}
```

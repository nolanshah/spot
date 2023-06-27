# spot

spot is like [blot](https://github.com/davidmerfield/blot) but on rails. Spot turns a folder into a static website. It can convert docx, md, pynb, txt, latex, rtf, and really any other format supported by pandoc into html. It's suitable for a blog, docsite, or whatever you want, but it only generates html -- how and where you host it is up to you.

spot is a perpetual prototype. I won't write unit tests or give support. Pull requests to add your own features are encouraged, I'm happy to merge in contributions!

## How to use spot

1. Go to the latest GitHub release, and download the binary for your platform.
2. Create a project folder (`mkdir my_new_project; cd my_new_project`)
3. Create spot scaffolding with `spot init --dir ./`
4. Build the project with `spot build --config ./config.yaml`
5. Start adding content to `./content`, update templates in `./templates`, and add static files to `./static`. Update the `config.yaml` as needed.
6. While you're iterating, use the watch & serve feature (and turn on debug logging) with `spot --debug build --config ./config.yaml --watch --addr :8080`
7. The `./dist` folder contains your built static website. You can publish it's contents however you want. 

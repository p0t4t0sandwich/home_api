gen:
	./tailwindcss -i ./src/web/assets/index.css -o ./public/styles.css --minify
	templ generate

/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./src/web/components/*.templ", "./src/api/modules/**/*.templ"],
  theme: {
    extend: {
      colors: {}
    },
  },
  plugins: [
    require('@tailwindcss/forms'),
    require('@tailwindcss/typography')
  ],
}

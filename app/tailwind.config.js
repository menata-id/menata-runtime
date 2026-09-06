/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    './internal/**/*.templ',
    './internal/**/*.go',
    './cmd/**/*.go',
  ],
  theme: {
    extend: {},
  },
  plugins: [],
}

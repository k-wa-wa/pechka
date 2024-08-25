/** @type {import('tailwindcss').Config} */

const konstaConfig = require("konsta/config")

module.exports = konstaConfig({
  content: ["./src/**/*.{js,jsx,ts,tsx}"],
  theme: {
    extend: {},
  },
  plugins: [],
})

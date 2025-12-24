/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      fontFamily: {
        display: ['"Space Grotesk"', 'sans-serif'],
        body: ['"Inter"', 'sans-serif'],
      },
      colors: {
        charcoal: '#0f0f14',
        sand: '#f7f2ea',
        ember: '#ff5f45',
        moss: '#00a86b',
      },
    },
  },
  plugins: [],
};

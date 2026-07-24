import { ref, onMounted } from 'vue'

const THEME_KEY = 'sh-theme'
export type Theme = 'dark' | 'light'

const theme = ref<Theme>('dark')

function apply(t: Theme) {
  theme.value = t
  document.documentElement.setAttribute('data-theme', t)
  try {
    localStorage.setItem(THEME_KEY, t)
  } catch {
    /* ignore */
  }
}

export function useTheme() {
  onMounted(() => {
    const saved = localStorage.getItem(THEME_KEY)
    if (saved === 'light' || saved === 'dark') apply(saved)
    else apply('dark')
  })

  function toggleTheme() {
    apply(theme.value === 'dark' ? 'light' : 'dark')
  }

  return { theme, toggleTheme, setTheme: apply }
}

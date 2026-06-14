import { createTheme } from '@mui/material/styles';

// Фирменная палитра приложения «Аграном»: Material 3, seed #2E7D32.
export const theme = createTheme({
  palette: {
    mode: 'light',
    primary: { main: '#2E7D32', dark: '#1B5E20', light: '#60AD5E' },
    secondary: { main: '#558B2F' },
    background: { default: '#F4F6F4', paper: '#FFFFFF' },
    text: { primary: '#1B2A1B', secondary: '#5A675A' },
  },
  shape: { borderRadius: 12 },
  typography: {
    fontFamily: '-apple-system, "Segoe UI", Roboto, Arial, sans-serif',
    h5: { fontWeight: 700 },
    h6: { fontWeight: 600 },
  },
  components: {
    MuiButton: {
      styleOverrides: {
        root: { textTransform: 'none', fontWeight: 600 },
      },
    },
    MuiPaper: {
      styleOverrides: {
        root: { backgroundImage: 'none' },
      },
    },
  },
});

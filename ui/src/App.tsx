import './App.css';
import Box from '@mui/material/Box';
import CssBaseline from '@mui/material/CssBaseline';
import { ThemeProvider } from '@mui/material/styles';
import { Routes, Route } from "react-router-dom"
import { Header } from './components';
import { ROUTES } from './pages';
import { MOLT_THEME } from './styles/theme';

function App() {
  return (
    <ThemeProvider theme={MOLT_THEME}>
      <CssBaseline />
      <Box className="App">
        <Header />
        <Routes>
          {ROUTES.map(item => {
            return <Route key={item.path} path={item.path} element={item.element} />
          })}
        </Routes>
      </Box>
    </ThemeProvider>
  );
}

export default App;

import './App.css';
import { Routes, Route } from "react-router-dom"
import { Header } from './components';
import { ROUTES } from './pages';

function App() {
  return (
    <div className="App">
      yo
      <Header />
      <Routes>
        {ROUTES.map(item => {
          return <Route key={item.path} path={item.path} element={item.element} />
        })}
      </Routes>
    </div>
  );
}

export default App;

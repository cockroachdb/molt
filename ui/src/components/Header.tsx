import AppBar from '@mui/material/AppBar';
import Box from '@mui/material/Box';
import CssBaseline from '@mui/material/CssBaseline';
import Toolbar from '@mui/material/Toolbar';
import Button from '@mui/material/Button';
import ButtonBase from '@mui/material/ButtonBase';

import { useNavigate, useLocation } from "react-router-dom";
import { DETAILED_FETCH_PATH, DETAILED_VERIFY_PATH, HOME_PATH, ROUTES } from '../pages/index';
import { neutral } from '../styles/colors';
import Logo from './Logo';

const navItems = ROUTES;

export default function Header() {
    const navigate = useNavigate();
    const location = useLocation();

    return (
        <Box sx={{ display: 'flex', pt: 5 }}>
            <CssBaseline />
            <AppBar sx={{ backgroundColor: neutral[900] }} component="nav">
                <Toolbar sx={{
                    display: "flex",
                    flexDirection: "row",
                    alignItems: "center",
                    gap: 2,
                }}>
                    <ButtonBase onClick={() => navigate(HOME_PATH)}><Logo type="cockroach-color-dark-bg" size="default" /></ButtonBase>
                    <Box sx={{ display: "flex", gap: 2 }}>
                        {navItems.filter(item => ![DETAILED_FETCH_PATH, DETAILED_VERIFY_PATH].includes(item.path)).map((item) => (
                            <Button onClick={() => navigate(item.path)} key={item.name}
                                sx={
                                    {
                                        color: location.pathname === item.path ? neutral[0] : neutral[400],
                                        textTransform: "none"
                                    }
                                }>
                                {item.name}
                            </Button>
                        ))}
                    </Box>
                </Toolbar>
            </AppBar>
        </Box>
    );
}

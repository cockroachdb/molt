import AppBar from '@mui/material/AppBar';
import Box from '@mui/material/Box';
import CssBaseline from '@mui/material/CssBaseline';
import Toolbar from '@mui/material/Toolbar';
import Button from '@mui/material/Button';

import { useNavigate, useLocation } from "react-router-dom";
import { ROUTES } from '../pages/index';
import { neutral } from '../styles/colors';
import Logo from './Logo';

const navItems = ROUTES;

export default function Header() {
    const navigate = useNavigate();
    const location = useLocation();

    return (
        <Box sx={{ display: 'flex', pt: 4 }}>
            <CssBaseline />
            <AppBar sx={{ backgroundColor: neutral[900] }} component="nav">
                <Toolbar sx={{
                    display: "flex",
                    flexDirection: "row",
                    alignItems: "center",
                    gap: 2,
                }}>
                    <Logo type="cockroach-color-dark-bg" size="default" />
                    <Box sx={{ display: "flex", gap: 2 }}>
                        {navItems.map((item) => (
                            <Button disableRipple onClick={() => navigate(item.path)} key={item.name}
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

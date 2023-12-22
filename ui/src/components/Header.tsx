import AppBar from '@mui/material/AppBar';
import Box from '@mui/material/Box';
import CssBaseline from '@mui/material/CssBaseline';
import IconButton from '@mui/material/IconButton';
import Toolbar from '@mui/material/Toolbar';
import Typography from '@mui/material/Typography';
import Button from '@mui/material/Button';
import { purple } from "@mui/material/colors";

import { useNavigate } from "react-router-dom";

import { APP_NAME } from '../utils/constants';
import { ROUTES } from '../pages/index';

interface Props {
    /**
     * Injected by the documentation to work in an iframe.
     * You won't need it on your project.
     */
    window?: () => Window;
}

const navItems = ROUTES;

export default function Header(props: Props) {
    const navigate = useNavigate();
    return (
        <Box sx={{ display: 'flex', pt: 5 }}>
            <CssBaseline />
            <AppBar sx={{ backgroundColor: purple[500] }} component="nav">
                <Toolbar>
                    <IconButton
                        color="inherit"
                        aria-label="open drawer"
                        edge="start"
                        sx={{ mr: 2, display: { sm: 'none' } }}
                    >
                    </IconButton>
                    <Typography
                        variant="h6"
                        component="div"
                        sx={{ flexGrow: 1, display: { xs: 'none', sm: 'block' } }}
                    >
                        {APP_NAME}
                    </Typography>
                    <Box sx={{ display: { xs: 'none', sm: 'block' } }}>
                        {navItems.map((item) => (
                            <Button onClick={() => navigate(item.path)} key={item.name} sx={{ color: '#fff', textTransform: "none" }}>
                                {item.name}
                            </Button>
                        ))}
                    </Box>
                </Toolbar>
            </AppBar>
        </Box>
    );
}

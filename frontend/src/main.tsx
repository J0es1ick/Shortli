import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import "./index.css";
import App from "./App.tsx";
import { ThemeProvider } from "./context/ThemeContext.tsx";
import { UserProvider } from "./context/UserContext.tsx";
import { LocaleProvider } from "./context/LocaleContext.tsx";

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <ThemeProvider>
      <LocaleProvider>
        <UserProvider>
          <App />
        </UserProvider>
      </LocaleProvider>
    </ThemeProvider>
  </StrictMode>,
);

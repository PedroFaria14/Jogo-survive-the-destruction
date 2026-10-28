import { lazy, Suspense, useCallback, useEffect, useState } from "react";
import { Home } from "./components/Home.jsx";
import { detectLanguage, STRINGS, translate } from "./i18n.js";

const Game = lazy(() => import("./Game.jsx"));

export default function App() {
  const [profile, setProfile] = useState(null);
  const [lang, setLang] = useState(detectLanguage);
  const home = useCallback(() => setProfile(null), []);
  const t = (key) => translate(STRINGS[lang], key);

  useEffect(() => {
    document.documentElement.lang = lang === "pt" ? "pt-BR" : "en";
    try {
      localStorage.setItem("lang", lang);
    } catch {
      /* Preferência opcional. */
    }
  }, [lang]);

  if (profile) {
    return (
      <Suspense
        fallback={
          <main className="app-loading" role="status">
            {t("loading.checking")}
          </main>
        }
      >
        <Game
          profile={profile}
          onHome={home}
          lang={lang}
          onLanguage={() => setLang((l) => (l === "pt" ? "en" : "pt"))}
        />
      </Suspense>
    );
  }
  return (
    <Home
      lang={lang}
      onLanguage={() => setLang(lang === "pt" ? "en" : "pt")}
      onPlay={setProfile}
    />
  );
}

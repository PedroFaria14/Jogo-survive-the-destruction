import { lazy, Suspense, useCallback, useEffect, useState } from "react";
import { Home } from "./components/Home.jsx";
import {
  detectLanguage,
  saveLanguagePreference,
  STRINGS,
  translate,
} from "./i18n.js";

const Game = lazy(() => import("./Game.jsx"));

export default function App() {
  const [profile, setProfile] = useState(null);
  const [lang, setLang] = useState(detectLanguage);
  const home = useCallback(() => setProfile(null), []);
  const toggleLanguage = useCallback(() => {
    const next = lang === "en" ? "pt" : "en";
    saveLanguagePreference(next);
    setLang(next);
  }, [lang]);
  const t = (key) => translate(STRINGS[lang], key);

  useEffect(() => {
    document.documentElement.lang = lang === "pt" ? "pt-BR" : "en";
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
          onLanguage={toggleLanguage}
        />
      </Suspense>
    );
  }
  return <Home lang={lang} onLanguage={toggleLanguage} onPlay={setProfile} />;
}

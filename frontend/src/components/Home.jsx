import { useEffect, useRef, useState } from "react";
import { STRINGS, translate } from "../i18n.js";
import heroImage from "../../assets/carregamento1.jpeg";
import "./home.css";

export function Home({ lang, onLanguage, onPlay }) {
  const [setup, setSetup] = useState(false);
  const [name, setName] = useState("");
  const [color, setColor] = useState(() => {
    try {
      return localStorage.getItem("ballColor") || "#f2b544";
    } catch {
      return "#f2b544";
    }
  });
  const titleRef = useRef(null);
  const nameRef = useRef(null);
  const t = (key) => translate(STRINGS[lang], key);
  useEffect(() => {
    if (setup) nameRef.current?.focus();
    else titleRef.current?.focus();
  }, [setup]);

  const submit = (event) => {
    event.preventDefault();
    if (window.matchMedia?.("(pointer: coarse)").matches) {
      document.documentElement.requestFullscreen?.().catch(() => {});
      window.screen?.orientation?.lock?.("landscape").catch(() => {});
    }
    try {
      localStorage.setItem("ballColor", color);
    } catch {
      /* Preferência opcional. */
    }
    onPlay({
      name: name.trim() || `Jogador${Math.floor(1000 + Math.random() * 9000)}`,
      color,
    });
  };

  return (
    <div className="home-scroll">
      <a className="skip-link" href="#home-main">
        {t("home.skip")}
      </a>
      <header className="home-header">
        <a
          className="home-brand"
          href="#home-main"
          onClick={() => setSetup(false)}
        >
          <span className="brand-orb" aria-hidden="true" /> SURVIVE
          <span className="brand-small">THE DESTRUCTION</span>
        </a>
        <nav aria-label={t("home.navigation")}>
          {!setup && <a href="#how-to-play">{t("home.how")}</a>}
          {!setup && <a href="#project">{t("home.projectLink")}</a>}
          <button
            className="language-button"
            onClick={onLanguage}
            aria-label={
              lang === "pt" ? "Switch to English" : "Mudar para português"
            }
          >
            {lang === "pt" ? "EN" : "PT"}
          </button>
        </nav>
      </header>

      <main id="home-main">
        <section className="home-hero" aria-labelledby="home-title">
          <div className="hero-copy">
            <p className="home-eyebrow">
              <span aria-hidden="true">✦</span> {t("home.eyebrow")}
            </p>
            <h1 id="home-title" tabIndex={-1} ref={titleRef}>
              {setup ? (
                t("home.setupTitle")
              ) : (
                <>
                  {t("home.title1")}
                  <br />
                  <em>{t("home.title2")}</em>
                </>
              )}
            </h1>
            <p className="hero-description">
              {setup ? t("home.setupDescription") : t("home.description")}
            </p>
            {setup ? (
              <form className="player-setup" onSubmit={submit}>
                <label htmlFor="player-name">{t("start.chooseName")}</label>
                <input
                  id="player-name"
                  ref={nameRef}
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  maxLength={20}
                  autoComplete="nickname"
                  placeholder={t("home.namePlaceholder")}
                />
                <div className="color-choice">
                  <label htmlFor="player-color">{t("start.ballColor")}</label>
                  <input
                    id="player-color"
                    type="color"
                    value={color}
                    onChange={(e) => setColor(e.target.value)}
                  />
                  <span
                    className="player-preview"
                    style={{ background: color }}
                    aria-hidden="true"
                  />
                </div>
                <button className="primary-button" type="submit">
                  {t("home.enter")} <span aria-hidden="true">↗</span>
                </button>
                <button
                  className="text-button"
                  type="button"
                  onClick={() => setSetup(false)}
                >
                  ← {t("home.back")}
                </button>
              </form>
            ) : (
              <>
                <div className="hero-actions">
                  <button
                    className="primary-button"
                    onClick={() => setSetup(true)}
                  >
                    {t("home.play")} <span aria-hidden="true">↗</span>
                  </button>
                  <a className="secondary-link" href="#project">
                    {t("home.projectLink")} ↓
                  </a>
                </div>
                <p className="home-note">{t("home.note")}</p>
                <dl className="home-facts">
                  <div>
                    <dt>01—05</dt>
                    <dd>{t("home.players")}</dd>
                  </div>
                  <div>
                    <dt>03</dt>
                    <dd>{t("home.lives")}</dd>
                  </div>
                  <div>
                    <dt>∞</dt>
                    <dd>{t("home.newArenas")}</dd>
                  </div>
                </dl>
              </>
            )}
          </div>
          <figure className="hero-scene">
            <img
              src={heroImage}
              width="1376"
              height="768"
              alt={t("home.imageAlt")}
              fetchPriority="high"
            />
            <div className="scene-label">
              <span aria-hidden="true">◉</span> {t("home.sceneLabel")}
            </div>
            <figcaption>{t("home.imageCaption")}</figcaption>
            <div className="scene-ticket">
              <span>THE FLOOR IS</span>
              <strong>NOT FOREVER.</strong>
            </div>
          </figure>
        </section>

        {!setup && (
          <>
            <section
              id="how-to-play"
              className="home-rules"
              aria-labelledby="rules-title"
            >
              <div className="section-heading">
                <p className="home-eyebrow">{t("home.rulesEyebrow")}</p>
                <h2 id="rules-title">{t("home.rulesTitle")}</h2>
              </div>
              <ol className="rules-list">
                {["move", "adapt", "survive"].map((rule, i) => (
                  <li key={rule}>
                    <span className="rule-number">0{i + 1}</span>
                    <div>
                      <h3>{t(`home.${rule}Title`)}</h3>
                      <p>{t(`home.${rule}Description`)}</p>
                    </div>
                  </li>
                ))}
              </ol>
              <div className="control-strip">
                <span>
                  <kbd>A</kbd>
                  <kbd>D</kbd> {t("home.move")}
                </span>
                <span>
                  <kbd>W / Space</kbd> {t("home.jump")}
                </span>
                <span>
                  <kbd>Shift</kbd> Dash
                </span>
                <span>{t("home.touch")}</span>
              </div>
            </section>
            <section
              id="project"
              className="home-project"
              aria-labelledby="project-title"
            >
              <div>
                <p className="home-eyebrow">{t("home.projectEyebrow")}</p>
                <h2 id="project-title">{t("home.projectTitle")}</h2>
              </div>
              <div>
                <p>{t("home.projectDescription")}</p>
                <ul className="tech-tags">
                  <li>Go</li>
                  <li>WebSocket</li>
                  <li>React + Canvas</li>
                  <li>PostgreSQL</li>
                </ul>
                <button
                  className="text-button"
                  onClick={() => {
                    setSetup(true);
                    document.querySelector(".home-scroll")?.scrollTo(0, 0);
                  }}
                >
                  {t("home.play")} ↗
                </button>
              </div>
            </section>
          </>
        )}
      </main>
      <footer className="home-footer">
        <span>SURVIVE THE DESTRUCTION</span>
        <span>{t("home.footer")}</span>
      </footer>
    </div>
  );
}

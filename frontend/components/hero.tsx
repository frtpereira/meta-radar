import React from "react";

export default function Hero({
    eyebrow,
    title,
    lede,
    meta,
    actions,
    className = "",
}: {
    eyebrow?: string;
    title: React.ReactNode;
    lede?: React.ReactNode;
    meta?: React.ReactNode;
    actions?: React.ReactNode;
    className?: string;
}) {
    return (
        <header className={`hero card ${className}`}>
            <div>
                {eyebrow ? <p className="eyebrow">{eyebrow}</p> : null}
                <h1>{title}</h1>
                {lede ? <p className="lede">{lede}</p> : null}
                {actions ?? null}
            </div>
            {meta ? <div className="hero__meta">{meta}</div> : null}
        </header>
    );
}

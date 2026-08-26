import React from "react";

export default function Card({
    heading,
    headingMeta,
    children,
    className = "",
}: {
    heading?: React.ReactNode;
    headingMeta?: React.ReactNode;
    children: React.ReactNode;
    className?: string;
}) {
    return (
        <section className={`card section ${className}`}>
            {heading ? (
                <div className="section__heading">
                    <div>
                        {typeof heading === "string" ? (
                            <>
                                {typeof headingMeta === "string" ? (
                                    <p className="eyebrow">{headingMeta}</p>
                                ) : null}
                                <h2>{heading}</h2>
                            </>
                        ) : (
                            heading
                        )}
                    </div>
                    {headingMeta && typeof headingMeta !== "string" ? (
                        <span className="muted">{headingMeta}</span>
                    ) : null}
                </div>
            ) : null}
            {children}
        </section>
    );
}

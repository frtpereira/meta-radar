import Link from "next/link";

export function NavigationBar() {
    return (
        <header className="site-nav">
            <div className="site-nav__inner">
                <Link className="site-nav__brand" href="/">
                    META Radar
                </Link>
                <nav className="site-nav__links" aria-label="Main">
                    <Link href="/">Home</Link>
                    <Link href="/tournaments">Tournaments</Link>
                    <Link href="/decklists">Decklists</Link>
                    <Link href="/matchups">Matchups</Link>
                </nav>
            </div>
        </header>
    );
}

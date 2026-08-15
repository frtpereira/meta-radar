Prompt:
I want you to look at limitlesstcg.com. Think about what's missing on it. What is it that could make an impact on a player that wants to know what's bombing, meta previews looking at Japan, etc. I am building a website that uses limitlesstcg's API, but has to deliver what's missing right now

Yes. I looked at what Limitless TCG currently provides, including the newer Labs functionality, and I think there is a **very clear opportunity**.

The key is: **don't build another Limitless clone. Build the layer that answers "what should I play?" rather than "what was played?"**

Limitless is already extremely good at archival tournament data, decklists, tournament standings, card data, and increasingly detailed matchup data. Labs can even combine metagame statistics across tournaments. ([Limitless TCG][1])

But there are still some very interesting gaps.

## The biggest opportunity: a "Meta Radar"

I'd make the homepage answer this question in **30 seconds**:

> **"If I have a tournament this weekend, what should I know?"**

Something like:

### 🔥 META RADAR — August 11

**Rising**

- Mega X + Y — ↑ 43%
- Deck X — ↑ 27%
- Deck Y — ↑ 18%

**Falling**

- Dragapult — ↓ 21%
- Deck X — ↓ 17%

**Dark horses**

- Deck A — 53.8% WR, 1.2% meta share
- Deck B — 55.1% WR, 0.8% meta share

**Japan signal**

> 🇯🇵 Deck X has gone from 2% → 14% over the last 10 days.

**Watch out**

> Deck X has a 63% win rate against the current top 5 decks.

That is substantially more actionable than another table of tournament results.

---

# 1. "What's bombing?" is actually a fantastic feature

This is probably the feature I'd build first.

Limitless tells you:

> Deck X — 47% win rate

Your site should tell you:

> 🔴 **Deck X is bombing**
>
> 17.4% meta share
> 44.2% expected win rate
> −8.1% performance vs popularity
> ↓ 32% over the last 14 days

The important metric is **popularity vs performance**.

You can calculate something like:

```text
Expected win rate
        vs
Actual win rate
        vs
Meta share
```

And classify decks:

| Type           | Meaning                            |
| -------------- | ---------------------------------- |
| 🟢 Rising Star | Low popularity + high WR           |
| 🔥 Meta        | High popularity + high WR          |
| ⚠️ Overhyped   | High popularity + mediocre WR      |
| 💀 Bombing     | High popularity + poor WR          |
| 🕵️ Sleeper     | Very low popularity + excellent WR |
| 📈 Emerging    | Rapidly increasing popularity      |
| 📉 Dying       | Rapidly decreasing popularity      |

This is **much more useful to a competitive player** than simply sorting by tournament finishes.

---

# 2. Separate "Popularity" from "Strength"

This is a huge one.

A deck being popular doesn't mean it's good.

You could create a 2D graph:

```text
                 HIGH WIN RATE
                      ↑
                      |
        SLEEPERS      |       META
                      |
    ------------------+------------------→ POPULARITY
                      |
       MEMES          |     OVERHYPED
                      |
                 LOW WIN RATE
```

Then every deck becomes a bubble.

For example:

**Dragapult**

> 24% meta share
> 49.1% win rate
> **Very popular / average performance**

Whereas:

**Random Rogue Deck**

> 1.7% meta share
> 56.3% win rate
> **Potential sleeper**

This gives players something they can't get just by browsing tournament results.

---

# 3. Japan should be a _prediction engine_, not merely another region

This is probably your biggest differentiator.

There's already a real player pain point here.

I found multiple recent discussions where players specifically say they have to use **Limitless + Play Limitless + Japanese City League results + PokecaBook** depending on where they are in the release/rotation cycle. ([Reddit][2])

That's a gigantic clue.

The problem isn't:

> "Where can I find Japanese decklists?"

It's:

> **"What is Japan telling me about my future meta?"**

Your site could create:

# 🇯🇵 Japan → 🌎 International Meta Forecast

For every Japanese archetype:

```text
JAPAN
──────────────

Deck X
7d ago       3.1%
Today        12.7%

Trend        ↑ +310%

────────────────

WESTERN FORECAST

Expected adoption:
████████░░ 78%

Confidence:
███████░░░ 71%
```

And then:

> **"Deck X is exploding in Japan. Historically, decks showing this pattern reach Western competitive play in ~2–4 weeks."**

That's _extremely_ valuable.

---

# 4. Build a "Japan Meta Preview"

I'd literally give this its own page.

### 🇯🇵 Japan Meta Preview

**Last 7 days**

| Deck   | Share |  Δ 7d |    WR | Signal |
| ------ | ----: | ----: | ----: | ------ |
| Deck A | 18.2% |  +42% | 54.1% | 🔥     |
| Deck B | 12.4% |   +8% | 52.7% | 🟢     |
| Deck C |  8.3% |  −14% | 48.2% | 📉     |
| Deck D |  4.1% | +187% | 55.4% | 🚨     |

Then:

### "What matters?"

> 🚨 **Deck D is the biggest emerging threat.**

> It appeared in 4.1% of Japanese lists this week, up from 1.4%. Its win rate is 55.4%, and 71% of its increase came from the last three days.

Then show:

### "What should Western players learn?"

- Cards appearing in new lists
- New tech choices
- New archetype combinations
- Matchups being targeted
- Cards disappearing

This turns raw Japanese data into **intelligence**.

---

# 5. Track individual cards, not just decks

This is another area I'd attack.

Instead of:

> "Deck X is increasing"

tell me:

> **"Why is Deck X increasing?"**

For every card:

### Card trend

```text
CARD X

Played in:
8.2% → 21.7% of decks

+164% over 14 days

Most common in:
Deck A     73%
Deck B     18%
Deck C      6%

Japan:
██████████ 34%

International:
████ 12%
```

Then:

> 🧠 **Emerging tech**

> Card X is appearing increasingly in Deck A, particularly in Japanese City Leagues.

This is **much more interesting than a card database**.

---

# 6. Detect "tech evolution"

This could become addictive for competitive players.

Imagine looking at a deck:

### Deck X — Evolution

**June 1**

```text
4 Card A
4 Card B
2 Card C
1 Card D
```

**June 15**

```text
4 Card A
4 Card B
2 Card C
2 Card D
```

**July 1**

```text
4 Card A
3 Card B
2 Card C
3 Card D
```

Then:

> **The meta is moving toward Card D.**

You can literally visualize decklists evolving over time.

Even better:

### Most common changes

```text
+1 Card D       43% of lists
+1 Boss        31%
-1 Card B      28%
+1 Tech C      19%
```

That is **player intelligence**.

---

# 7. "What are people preparing for?"

This is a very interesting derived metric.

Suppose:

- Deck A is 25% of the meta
- Deck B is 15%
- Deck C is 10%

And suddenly:

> Cards specifically good against Deck A are appearing in 35% more lists.

Your site can say:

## 🚨 The meta is preparing for Deck A

> Tech cards targeting Deck A have increased 64% over the last two weeks.

That's a much stronger signal than:

> "Deck A is still #1."

You are essentially measuring **metagame adaptation**.

---

# 8. Matchup matrix, but actually useful

Limitless Labs already has matchup data, so I wouldn't copy it. Labs can show deck-specific matchup results in individual tournaments. ([Limitless Labs][3])

Instead, aggregate it.

Something like:

|           | Dragapult | Gardevoir | Zoroark | Deck X |
| --------- | --------: | --------: | ------: | -----: |
| Dragapult |         — |       48% |     54% |    42% |
| Gardevoir |       52% |         — |     46% |    57% |
| Zoroark   |       46% |       54% |       — |    51% |
| Deck X    |       58% |       43% |     49% |      — |

But add:

### **Meta-adjusted win rate**

A deck with:

```text
55% WR
```

isn't necessarily better than a deck with:

```text
52% WR
```

if the 55% deck farmed terrible rogue decks.

Calculate its expected win rate against the **current metagame distribution**.

That's a much more meaningful number:

> **Expected tournament WR: 54.8%**

---

# 9. "If the tournament were tomorrow..."

This could be one of your killer features.

User chooses:

> Region: Europe
> Format: current
> Tournament: 1000 players

Your system produces:

# 🎯 Tournament Simulator

### Expected field

```text
Dragapult          19%
Zoroark             9%
Gardevoir           8%
Deck X              7%
Deck Y              6%
...
```

Then:

### Best decks for this field

```text
1. Deck A     54.7%
2. Deck B     54.2%
3. Deck C     53.8%
```

And:

### Best rogue

> 🕵️ Deck X

> Only 2.4% expected popularity but 55.1% expected win rate against the projected field.

This turns your website from a **database into a decision tool**.

---

# 10. "What should I play?"

Eventually you can have:

## 🧭 Deck Finder

Ask:

> "I hate playing Dragapult."

or filters:

```text
☑ Competitive
☑ Low popularity
☐ Easy to play
☑ Good into Dragapult
☑ Good into Gardevoir
☐ Cheap
```

And return:

> ### 🥇 Deck X
>
> **Projected WR: 54.3%**
>
> Strong against:
>
> - Dragapult: 58%
> - Gardevoir: 61%
> - Zoroark: 54%
>
> Weak against:
>
> - Deck Y: 41%
>
> Meta share: 2.8%
>
> **Why:** Strong matchup spread while being significantly underplayed.

That is something I'd personally use over another tournament browser.

---

# 11. A "Meta Timeline"

This would also be fantastic.

Let me drag a timeline:

```text
JAN ─── FEB ─── MAR ─── APR ─── MAY ─── JUN ─── JUL ─── AUG
        ↑              ↑                    ↑
      Deck A         Deck B               Deck C
```

Click a date and see:

### Meta — June 14

```text
1. Dragapult      21%
2. Gardevoir      15%
3. Zoroark         9%
```

Then:

### Meta — July 14

```text
1. Dragapult      15%
2. Zoroark        13%
3. Deck X         11%
```

This would let players understand **how the meta got here**.

---

# 12. "Why did this deck win?"

Another opportunity is post-tournament analysis.

Instead of simply:

> 1st — Deck X

Generate:

### 🏆 Why Deck X won

**Metagame position**

> Deck X represented only 4.2% of the field.

**Matchup advantage**

> 61% expected WR against the top 5 decks.

**Innovation**

> 73% of the winning list's tech configuration was uncommon among other Deck X lists.

**Path to victory**

```text
R5  ✓ Deck A
R6  ✓ Deck B
R7  ✓ Deck C
R8  ✓ Deck A
R9  ✓ Deck D
```

You can make tournament results **tell a story**.

---

# 13. Player trends

Limitless has player/deck information, but there's another interesting layer:

### Players to watch

Not just:

> World Champion X

But:

> **Player X has been testing Deck A for 4 weeks.**

Or:

> Player X switched from Deck A → Deck B immediately after the Japanese meta shifted.

You could track:

- player deck choices
- archetype switching
- repeated top finishes
- innovation
- tech choices

And eventually:

## 🧠 "What are top players playing?"

That could become a very useful signal.

---

# 14. The really cool one: "Meta Momentum"

I'd create a single metric for every archetype:

### Meta Momentum™ _(your own metric)_

Something like:

```text
Momentum =
Popularity growth
×
Win rate
×
Tournament quality
×
Recency
×
Japan signal
```

Not necessarily literally that formula, but conceptually.

Then:

### 🔥 META MOMENTUM

| Deck      | Momentum |
| --------- | -------: |
| Deck X    |       94 |
| Deck A    |       81 |
| Deck B    |       76 |
| Dragapult |       41 |
| Deck C    |       19 |

And:

> **Deck X is currently the fastest-growing competitive archetype.**

That's the kind of thing people would screenshot/share.

---

# 15. Don't underestimate "news"

The data itself can generate news automatically.

Instead of needing writers:

> ### 🚨 Meta Alert
>
> **Deck X has exploded in Japan**
>
> Its share increased from 3.2% → 11.8% over the past week.
>
> It has a 55.7% win rate across 412 games.
>
> The biggest change is the addition of Card Y.
>
> **Western implication:** expect this archetype to appear at upcoming Regionals.

Now your site has a constantly changing homepage.

---

# The architecture I'd build

Since you're using the Limitless API, I'd put your application **on top of** their data rather than trying to replace their database.

```text
                 Limitless API
                      │
                      ▼
             ┌─────────────────┐
             │ Data Ingestion   │
             └────────┬────────┘
                      │
                      ▼
             ┌─────────────────┐
             │ Normalization    │
             │                 │
             │ Deck archetypes │
             │ Cards           │
             │ Tournaments     │
             │ Regions         │
             └────────┬────────┘
                      │
                      ▼
             ┌─────────────────┐
             │ Analytics Engine │
             │                 │
             │ Win rates       │
             │ Meta share      │
             │ Trends          │
             │ Matchups        │
             │ Card trends     │
             │ Japan signals   │
             └────────┬────────┘
                      │
                      ▼
             ┌─────────────────┐
             │ Your API        │
             └────────┬────────┘
                      │
          ┌───────────┼───────────┐
          ▼           ▼           ▼
       Meta Radar   Japan       Deck Finder
                    Forecast
```

The important part is that **your database stores historical snapshots**.

Don't just ask Limitless:

> "What's the current meta?"

Store:

```text
2026-08-01
Deck A: 18.2%
Deck B: 12.4%
Deck C: 8.7%

2026-08-02
Deck A: 19.1%
Deck B: 11.8%
Deck C: 9.2%

2026-08-03
...
```

Then you can calculate trends that Limitless itself doesn't need to expose.

---

# If I were building this project

I'd prioritize it like this:

### 🥇 MVP

**Meta Radar**

- Current meta share
- Win rate
- Meta-adjusted win rate
- Rising/falling
- Overhyped/bombing
- Sleeper decks
- Trend charts

### 🥈

**Japan Meta Preview**

- Japanese tournament aggregation
- 7/14/30-day trends
- Emerging decks
- Emerging cards
- Japan → International comparison

### 🥉

**Deck Evolution**

- Historical decklists
- Card inclusion trends
- "Cards being added/removed"
- Tech evolution

### 4

**Matchup Intelligence**

- Aggregate matchup matrix
- Meta-adjusted matchup
- "Best decks against current meta"

### 5

**Tournament Simulator**

- User selects tournament/region
- Predict field
- Calculate best decks against predicted field

### 6

**Alerts**

> 🔥 Deck X is exploding

> 🚨 Card Y has doubled in usage

> 🇯🇵 Japan signal detected

> 📉 Deck X is collapsing

---

## And I think there's one particularly important insight

**Don't compete with Limitless on breadth.**

Limitless is already the place people go to answer:

> "Show me the tournament."

Your product should answer:

> **"Tell me what the tournament data means."**

That's the gap.

And the evidence that this is useful is already there: players currently bounce between Limitless, Play Limitless, Japanese City League results and PokecaBook because the information they need is fragmented across those sources. ([Reddit][2]) PokecaBook, for example, explicitly focuses on aggregating Japanese competitive deck results and publishes its own current tier analysis. ([PokecaBook][4])

Meanwhile, Limitless itself has moved toward deeper analytics with Labs, including complete tournament metagames and matchup win rates. ([Limitless TCG][5])

So **the next layer is interpretation + prediction**.

If you want, I can next go one level deeper and design the **actual MVP for this project** — database schema, API endpoints, algorithms for "bombing/rising/sleeper" detection, and the exact homepage/dashboard I'd build.

[1]: https://limitlesstcg.com/tournaments?utm_source=chatgpt.com "Tournaments – Limitless"
[2]: https://www.reddit.com/r/PTCGL/comments/1svug8r/where_can_i_find_meta_deck_lists_for_latest/?utm_source=chatgpt.com "Where can I find meta deck lists for latest expansión?"
[3]: https://labs.limitlesstcg.com/0014/decks/miraidon-ex/matchups?utm_source=chatgpt.com "Miraidon • Regional Championship Toronto – Limitless Labs"
[4]: https://pokecabook.com/?utm_source=chatgpt.com "PokecaBook"
[5]: https://limitlesstcg.com/limitless-labs-launch?utm_source=chatgpt.com "Limitless Labs Launch – Limitless"

Prompt:
What's the revenue model for this? I'm thinking about implementing ads (banners) and maybe a premium subscription, but I want to make sure that the free user also gets great value, and maybe offer AI insights to pro plan users.

Yes — I think **ads + premium is the right model**, but I would be careful about what you put behind the paywall.

The business should fundamentally be:

> **Free = "What's happening?"**
> **Pro = "What should I do about it?"**

That distinction gives you a genuinely useful free product while creating a reason to subscribe.

One important thing first: Limitless explicitly supports public-facing projects using its tournament API, and says API keys are intended for public-facing projects with a legitimate use case. I'd make sure your monetization/data usage is consistent with whatever access they approve for your project before building the business around it. ([Limitless Docs][1])

---

# 💰 I'd use 3 revenue streams

### 1. Free users → ads

### 2. Pro users → subscription

### 3. Later → affiliate/sponsorship revenue

And I'd make **Pro primarily about decision support, personalization and AI**, rather than arbitrarily locking basic statistics.

---

# 🆓 Free should be REALLY good

This is important.

If someone visits your site and sees:

> "You need Pro to see the meta."

They'll just go back to Limitless.

Instead, give them something **better than Limitless for free**.

I'd make the entire **Meta Radar** free.

For example:

# 🔥 Current Meta

| Deck      | Share | Win rate | Trend | Signal |
| --------- | ----: | -------: | ----: | ------ |
| Dragapult | 21.4% |    51.8% |     ↓ | 🟡     |
| Zoroark   | 13.2% |    54.1% |     ↑ | 🟢     |
| Gardevoir |  9.8% |    52.9% |     → | 🟢     |
| Deck X    |  2.1% |    57.2% |    ↑↑ | 🔥     |

And:

> **🔥 Deck X is the current sleeper.**

That should cost **€0**.

Why?

Because this is your acquisition funnel.

Someone searches Google:

> "Pokemon TCG meta"

They land on your site.

They see something genuinely useful.

Then:

> **Want to know what you should play against this meta?**

That's where Pro starts.

---

# 🧠 Pro should answer questions

I'd make the Pro pitch something like:

> **Stop looking at tournament results. Start making better tournament decisions.**

And then give them:

### Free

- Current meta
- Win rates
- Meta share
- Rising/falling decks
- Basic matchup data
- Tournament results
- Decklists
- Japan trends
- Basic card trends
- Historical charts

### Pro

- AI Meta Analyst
- Personalized deck recommendations
- Tournament simulator
- Advanced matchup analysis
- Meta forecasting
- Japan → International predictions
- Deck optimization
- Tech recommendations
- Custom alerts
- Saved decks
- Personal testing history
- Advanced filters
- No ads

That's a **very clean division**.

---

# 🤖 AI is actually where I'd charge

But **don't make "AI chatbot" the product**.

That's too easy to copy.

Instead, AI should sit on top of your proprietary analytics.

For example, a user asks:

> "I have a Regional next weekend. What should I play?"

Your AI shouldn't just answer based on its Pokémon knowledge.

It should have access to your data:

```text
Current meta
+
Recent tournament results
+
Japan trends
+
Matchup matrix
+
Deck popularity
+
Card trends
+
User's preferences
```

Then produce:

> **I'd currently recommend Deck X for you.**
>
> Here's why:
>
> - 54.8% projected win rate
> - Only 3.2% projected field share
> - 57% against the current #1 deck
> - Rising in Japan but not yet adopted heavily in Europe
> - Your preferred archetype matches its play style
>
> **Main concern:** Deck Y is becoming increasingly popular and is a poor matchup.

That's something people could plausibly pay for.

---

# 💎 I'd actually make Pro around "Tournament Prep"

This is your killer product.

Imagine a Pro user has:

> **Barcelona Regional — Saturday**

They click:

# 🎯 Prepare for Tournament

And get:

### Expected meta

```text
Dragapult       18%
Zoroark         12%
Gardevoir        9%
Deck X           7%
Deck Y           5%
...
```

### Best decks against projected field

```text
🥇 Deck A     55.1%
🥈 Deck B     54.7%
🥉 Deck C     54.3%
```

### Biggest threats

> ⚠️ Deck X is gaining popularity unusually quickly.

### Emerging tech

> Card Y has appeared in 31% of Deck A lists this week, compared with 12% last week.

### Your deck

User selects their deck.

> **Your Deck A**
>
> Projected WR: **53.2%**
>
> Expected performance:
>
> 🟢 Strong vs Dragapult
> 🟢 Strong vs Gardevoir
> 🟡 Even vs Zoroark
> 🔴 Weak vs Deck X
>
> **Recommendation:** Add 1× Card Y.

Then:

> **Ask AI about this analysis →**

That's a premium product.

---

# 💵 What would I charge?

I'd start surprisingly cheap.

### Free

€0

### Pro

**€4.99/month**

or

**€39.99/year**

That's about €3.33/month when paid annually.

I'd eventually experiment with:

**€6.99/month / €59.99/year**

but I wouldn't start there.

You're competing for discretionary spending from Pokémon players, not enterprise customers.

---

# And I'd strongly consider a tournament pass

This could be surprisingly good.

Imagine someone doesn't normally use the site.

They have:

> EUIC next weekend.

Instead of subscribing:

### Tournament Pass

**€2.99**

Unlock Pro tournament preparation for 7 days.

That gives them:

- Tournament simulator
- Meta forecast
- AI deck analysis
- AI matchup analysis
- Japan preview
- Advanced deck optimization
- No ads

Then after the tournament:

> **Enjoying Pro? €39.99/year →**

This is especially useful because Pokémon TCG usage is **event-driven**.

Someone may not want a year-round subscription.

But they absolutely might pay €3 before a Regional.

---

# 📢 Ads

I would absolutely implement ads.

But **don't turn it into an ad farm**.

Your audience is relatively niche, so display advertising alone probably won't be amazing.

Google defines ad RPM as revenue per 1,000 impressions, so your economics are basically:

> traffic × pageviews × RPM. ([Google Support][2])

And community reports from gaming/general sites suggest RPM can be quite variable and often relatively low, so I wouldn't design the business assuming ads will carry it. ([Reddit][3])

For example, purely hypothetically:

```text
100k pageviews/month
× €3 RPM
----------------
€300/month
```

That's nice.

But it's not a business by itself.

Whereas:

```text
100k monthly users
× 2% Pro conversion
× €5/month
----------------
€10,000/month
```

Now you're talking.

Even:

```text
50k users
× 1%
× €5
=
€2,500/month
```

plus ads and affiliates.

---

# 🛒 The sneaky third revenue stream: card affiliates

This is **very relevant to your audience**.

Suppose your AI says:

> Add 2× Card X.

Right underneath:

> **Need Card X?**
>
> [Find Card X]

You can potentially monetize purchases through card-marketplace affiliate programs.

Limitless itself already exposes a TCGplayer affiliate link, so this is an established monetization pattern in this ecosystem. ([Limitless Tournament Platform][4])

You could eventually have:

> **Your deck is missing 4 cards**
>
> Estimated cost: €18.40
>
> [Find cards]

And earn affiliate commission.

That is much more naturally aligned with the product than generic banner advertising.

---

# 🤝 Sponsorships could eventually beat ads

Once you have meaningful traffic:

> **Meta Radar sponsored by [TCG store]**

or:

> **Japan Meta Report — sponsored by [store]**

Potential advertisers:

- Card stores
- Card marketplaces
- Deck accessories
- Sleeves
- Deck boxes
- Playmats
- Tournament organizers
- Coaching services
- TCG apps

And unlike generic display ads, you're selling access to a **very specific audience**:

> Competitive Pokémon TCG players.

That's valuable.

---

# 🚨 But here's something I would NOT do

Don't make this:

### Free

> 5 decks

### Pro

> 100 decks

That's boring.

Or:

### Free

> Meta: 50%

### Pro

> Meta: 100%

Also bad.

People should feel:

> **"This website is incredibly useful even if I never pay."**

Then Pro should make them think:

> **"Holy shit, this would actually help me prepare for tournaments."**

That's the psychological difference.

---

# My ideal product structure

I'd actually make the website look like this:

```text
┌─────────────────────────────────────────────┐
│              META RADAR                     │
│                                             │
│  🔥 Rising   💀 Bombing   🕵️ Sleepers      │
│                                             │
│  Current Meta                               │
│  Matchups                                   │
│  Japan                                      │
│  Card Trends                                │
└─────────────────────────────────────────────┘

FREE
─────────────────────────────────────────────

             ↓

┌─────────────────────────────────────────────┐
│             TOURNAMENT PREP                 │
│                                             │
│  Choose tournament                          │
│  Choose deck                                │
│  Simulate meta                              │
│  Optimize list                              │
│  Prepare matchups                           │
│                                             │
│             🔒 PRO                           │
└─────────────────────────────────────────────┘

             ↓

┌─────────────────────────────────────────────┐
│              AI ANALYST                     │
│                                             │
│ "What should I play?"                       │
│ "How should I tech my deck?"                │
│ "What is Japan discovering?"                │
│ "What will the meta look like in 3 weeks?" │
│                                             │
│             🔒 PRO                           │
└─────────────────────────────────────────────┘
```

---

# One feature I'd REALLY put behind Pro

## 🔔 Meta Alerts

This is potentially worth more than the chatbot.

Let users follow:

- Decks
- Cards
- Archetypes
- Matchups
- Players
- Regions

Then:

> 🔔 **Meta Alert**
>
> Zoroark has increased from 8.2% → 14.7% in European tournaments over the past 10 days.

Or:

> 🇯🇵 **Japan Alert**
>
> Card X usage has increased 240% in Japanese City League lists.

Or:

> 🚨 **Your Deck**
>
> Deck Y has become your deck's worst matchup after the latest meta shift.

**That's recurring value.**

A chatbot gets used once.

An alert system keeps the subscription alive.

---

# I'd actually structure the business like this

| Revenue                      | Free    | Pro       |
| ---------------------------- | ------- | --------- |
| Display ads                  | ✅      | ❌        |
| Meta Radar                   | ✅      | ✅        |
| Tournament results           | ✅      | ✅        |
| Decklists                    | ✅      | ✅        |
| Basic trends                 | ✅      | ✅        |
| Japan meta                   | ✅      | ✅        |
| Advanced Japan forecasting   | ❌      | ✅        |
| Advanced matchup analysis    | ❌      | ✅        |
| Tournament simulator         | ❌      | ✅        |
| Deck optimizer               | ❌      | ✅        |
| AI Analyst                   | ❌      | ✅        |
| Personalized recommendations | ❌      | ✅        |
| Meta alerts                  | Limited | ✅        |
| Saved decks                  | 1–2     | Unlimited |
| No ads                       | ❌      | ✅        |

---

## And I'd add one important product principle

**Don't sell AI tokens. Sell outcomes.**

I wouldn't do:

> 100 AI messages/month

Users don't care about tokens.

Sell:

> **Tournament Preparation**

> **Meta Intelligence**

> **Deck Optimization**

> **Personalized Alerts**

The AI is just the mechanism underneath.

And there's a nice strategic reason for this: Limitless is already investing in deeper analytics through Labs, so simply making prettier charts isn't enough differentiation. ([Limitless Labs][5]) Your moat should become **historical analysis + forecasting + personalization + recommendations**.

If you execute that well, I think the interesting business isn't actually _"a Pokémon stats website with ads."_

It's:

> **"Bloomberg Terminal for competitive Pokémon TCG."**

Free users get the **Bloomberg homepage**.
Pro users get the **terminal**.

That's the model I'd build around.

[1]: https://docs.limitlesstcg.com/developer.html?utm_source=chatgpt.com "Developer Guides | Limitless Docs"
[2]: https://support.google.com/adsense/answer/112032?hl=pt&utm_source=chatgpt.com "RPM do anúncio - Google AdSense Ajuda"
[3]: https://www.reddit.com/r/Adsense/comments/1qsxa8e/new_website_low_rpm/?utm_source=chatgpt.com "New Website Low RPM"
[4]: https://play.limitlesstcg.com/help?utm_source=chatgpt.com "Help | Limitless"
[5]: https://labs.limitlesstcg.com/?utm_source=chatgpt.com "Limitless Labs • In-depth Pokémon TCG tournament data"

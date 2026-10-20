# 0039 — Daily needs follow a bounded goal

- **Status:** Accepted (supersedes [0028](0028-daily-needs-not-diets.md))
- **Date:** 2026-10-20
- **Epic:** EP-191

## Context

ADR 0028 made the daily needs reference intakes for keeping one's weight and ruled out any weight-loss or gain target: "a deficit is a clinical decision". The owner asked (2026-10-20) that the macronutrients Axon proposes, and the bars that measure the day against them, follow the person's dietary goal, which Axon must therefore let them state. Numeric targets drive restriction; the harm 0028 guarded against is real, and the eating-disorder and under-18 cases do not change.

## Decision

The health profile carries a **goal**: `maintain` (the default and what every existing profile keeps), `lose`, or `gain`. The needs are worked out as before for maintenance, then moved **within bounds, in code, never by a model**:

- **lose**: energy 15 % under maintenance, **never under the basal rate and never under 1,200 kcal** (the floor the agents' safety guard already keeps); when the share would cross the floor, the energy stops at the floor and the explanation says so. Protein rises to keep muscle: 0.4 g/kg more, at least 1.6 and at most 2.0 g/kg (ISSN position stand).
- **gain**: energy 10 % over maintenance; protein as for lose.
- Fat stays 30 % of the energy, carbohydrates the rest (never negative).

Every figure still says how it was worked out, with the pace in words (about 0.4 kg a week for a 2,760 kcal maintenance). The warning to check with a registered dietitian is always shown; there are still no numbers before 18 or during pregnancy or breastfeeding. The Chef & Nutritionist reads the same needs and still never sets targets of its own; the "no daily target under 1,200 kcal" guard stays. The goal is health data under the `health_data` consent, exported with it and deleted with it.

## Alternatives

- Keep 0028 and let a goal only change the split (more protein, same energy): would not answer the request, since the bars would not follow a loss or gain.
- Let the person type their own energy and macros: honest, but it invites the numbers 0028 feared, with no bound but the 1,200 kcal floor.
- A target weight and date, with the deficit worked out from them: more precise, more pressure; the bounded share gives a gentle pace without a countdown.

## Consequences

People who want to lose or gain see needs that follow that goal, at a gentle pace they cannot sharpen in Axon, and the day's bars measure against them. The floor means a small sedentary person sees the share stop at 1,200 kcal with the reason. Reversing is one migration (the column's default is `maintain`) and the removal of the switch in `health.Calc`.

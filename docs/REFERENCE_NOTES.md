# Xenon 2 reference notes

## Target edition

The supplied reference is the 1989 Amiga floppy edition of *Xenon 2:
Megablast*. It is not the CDTV, DOS, Mega Drive or Master System edition.
Differences between these releases must not be used to fill gaps in the
Amiga game's rules.

The local disk image is a Black Monks trained release. Trainer settings are
not normal game rules. Comparisons of lives, money, shields, enemy damage
and level progression require the trainer to be disabled.

## Rules documented by the original manual

The [archived Amiga/ST instruction manual](https://www.gamesdatabase.org/Media/SYSTEM/Commodore_Amiga/Manual/formated/Xenon_2-_Megablast_-_1989_-_Image_Works.htm)
describes these features:

- Five levels, with a final guardian in each and additional large enemies
  halfway through levels two to five.
- Three starting ships and six restart positions per level.
- Two shop visits per level: halfway through and at the end.
- A shield absorbs damage; the next hit after its exhaustion destroys the ship.
- Scenery obstructs movement, but damage comes from becoming trapped and
  crushed by the moving screen. Pulling back at the bottom briefly reverses scrolling.
- Shop selection quotes a price before a separate buy or sell confirmation.
- Super Nashwan lasts ten seconds. Dive lasts up to ten seconds and can be
  cancelled. Ordinary firing and the Autofire upgrade are distinct.
- Initials are entered for a ten-entry high-score table.

The manual's description of horizontal scrolling is inconsistent with the
game itself. The Amiga playfield scrolls vertically, with lateral camera movement.
The manual also describes only part of the equipment catalogue; it is not a
complete list of weapon slots, tiers or prices.

## Contemporary observations

[Original 1989 magazine reviews, preserved in transcription](https://www.amigareviews.leveluphost.com/xenon2.htm),
identify the level themes as prehistoric marine life, insects, fish,
dinosaurs and a mechanical future. They describe slight ship inertia,
animated weapon attachments, retaining equipment after death, and a choice
between in-game music and sound effects.

Printed catalogues disagree about some prices, including the cannon and
large mine. The supplied game's item records and purchase routines are the
authority for prices, compatibility, refunds and repeated upgrades.

## Directly observed menu data

The unpacked local game data contains the four choices `1 PLAYER GAME`,
`2 PLAYER GAME`, `MUSIC ON` and `MUSIC OFF`. It also contains ten initial
high-score entries and a continue countdown. These observations establish
the menu inventory, not the implementation of alternating turns or the
number of available continues.

## Timing and 60 Hz presentation

No verified simulation frequency has yet been established for this disk.
PAL video refresh, game logic frequency and rendered frame frequency are
different quantities. An internet video labelled 50 FPS does not establish
that the game's simulation updates fifty times per second.

The supplied unpacked program also contains an additional clock-counter
increment in a routine called from the vertical-blank interrupt. Its wait
routine uses a configurable counter threshold. The trained release's
effective cadence therefore cannot be treated as an unmodified retail
reference until those changes have been separated from the original code.

The game reads the Amiga beam-position registers and joystick hardware.
The beam wait and its main-loop callers must be traced before choosing the
Go simulation clock. Ship motion, scroll speed, enemy paths, firing delays,
animation, invulnerability and timed equipment must preserve elapsed game
time. Rendering at 60 Hz can interpolate positions between verified logic
states without multiplying these rates.

## Video references

- [World of Longplays, Amiga recording (2008)](https://www.youtube.com/watch?v=DCAUTrgso8k).
  The recorder explicitly reports emulator autofire, save states and removed
  loading pauses. It is useful for observing graphics, shops and progression,
  but not for inferring the original trigger rate or uninterrupted pacing.
- [AL82, Amiga recording (2021)](https://www.youtube.com/watch?v=zGZ7VQzJTvc).
  The listed level starts are 04:38, 12:38, 21:07, 29:17 and 40:17. A CRT
  filter is applied, so its output is not a palette or pixel reference.

## Comparison coverage

A faithful conversion needs comparisons beyond a successful first-level run:

| Area | Required comparison |
| --- | --- |
| Presentation | Introduction, title, credits, menu, high scores, loading transitions, HUD, shops and ending |
| Movement | Acceleration, release drift, every speed tier, map boundaries, wall sliding, crushing and reverse-scroll limits |
| Levels | Both sections of all five levels, all restart points, wave order, paths, fixed enemies and power-up placement |
| Weapons | Every attachment, tier, slot, compatibility rule, projectile pattern, damage rule and timed effect |
| Guardians | Active parts, weak points, vulnerability transitions, attacks, destruction order and rewards |
| Economy | Cash drops, collection, quotes, purchase refusal, sale refunds and shop-exit balances |
| Survival | Shield damage, invulnerability, death, retained equipment, checkpoint recovery, continues and alternating turns |
| Sound | Introduction and gameplay music, shop speech, effects, music/effects selection and replay timing |
| Completion | Final victory sequence, high-score insertion, initials and return to the attract loop |

Exact enemy records, collision masks, score awards, item prices and timing
constants remain subject to direct disk analysis. A familiar-looking
enemy shape or a plausible movement curve does not verify those rules.

# Age rating and export compliance

## What the app shows

- Posters, backdrops, synopses, cast and **trailers** of any movie or show listed by TMDB (trending, search, people,
  suggestions). The server asks TMDB with `include_adult=false`, so pornographic titles are excluded, but regular
  R-rated / TV-MA titles (horror, violence, sexual themes, drugs) appear with their posters and trailers.
- Trailers play in the app (Teasers feed, iOS `TeaserVideo`, Android `FilePlayer`) and external links open in the
  system browser or the matching app. No in-app web browser.
- **User-generated content**: comments on issues and decline reasons, visible to other users **of the same private
  server** only. No public feed, no chat with strangers. Moderation is done by the server admin (can delete issues).
- No gambling, no purchases, no location, no ads, no contests.

Content is not curated by the developer; it mirrors a film database, like IMDb or TMDB apps (rated 12+/Teen).

## App Store (questionnaire)

| Question | Answer |
|---|---|
| Cartoon or fantasy violence | Infrequent/Mild |
| Realistic violence | Infrequent/Mild |
| Prolonged graphic or sadistic realistic violence | None |
| Profanity or crude humor | Infrequent/Mild |
| Mature or suggestive themes | Infrequent/Mild |
| Horror/fear themes | Infrequent/Mild |
| Medical or treatment information | None |
| Alcohol, tobacco, or drug use or references | Infrequent/Mild |
| Sexual content or nudity | Infrequent/Mild (poster art / trailers of mainstream films) |
| Graphic sexual content and nudity | None |
| Simulated gambling / contests / loot boxes | None |
| Unrestricted web access | No |
| User-generated content | Yes (comments within a private server); messaging/chat between users: No |
| Age assurance / parental controls | No |

Expected result: **13+** (under the 2025 Apple age-rating scale; 12+ on the old one). Answering "Frequent/Intense"
for mature themes would give 16+, which is excessive for a catalog app whose user chooses what to open.

## Google Play (IARC questionnaire)

- Category: **Reference, News, or Educational** is wrong; choose **"All Other App Types"** (or "Entertainment" if
  offered). Do not choose "Game".
- Violence, sexuality, language, controlled substances: "Does the app contain …?" → answer **Yes** only where the
  questionnaire asks about *references or depictions in content such as trailers/images*, at mild level, matching the
  App Store answers above.
- Users can interact / exchange content: **Yes** (comments within a private server). Shares location: No.
  Digital purchases: No. Unrestricted internet: No.

Expected result: **Teen / PEGI 12 / USK 12**.

Also in Play Console → App content: Ads **No**; Target audience **18+** (or 13+ minimum, never children: avoids the
Families policy); News app **No**; Government app **No**; Financial features **None**; Health **No**.

## Export compliance

- `INFOPLIST_KEY_ITSAppUsesNonExemptEncryption = NO` is already set in `ios/Tipsarr/Tipsarr.xcodeproj/project.xcproj`.
  The app only uses HTTPS (URLSession), the Keychain and Apple's CryptoKit/Security APIs through the OS: exempt.
  App Store Connect will not ask the question for each build.
- Google Play: no export questionnaire in the Console; the app uses only OS-provided TLS and Android Keystore AES
  (exempt under EAR 740.17(b)(1) / mass-market). Nothing to file.

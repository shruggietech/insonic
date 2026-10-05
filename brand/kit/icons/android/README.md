# Android icons

Copy the `app/src/main/res` tree into an Android application and upload the separate Play image in Play Console. Declare `android:icon="@mipmap/ic_launcher"` in the app manifest; if declaring `android:roundIcon`, point it to a separately qualified round resource. Verify the packaged APK manifest and resources along with these source files.

| Path | Use |
|---|---|
| `app/src/main/res` | Legacy density PNGs and adaptive foreground, background, and optional monochrome resources |
| `play-store/google-play-512.png` | Full-square Google Play listing artwork; use only in Play Console |

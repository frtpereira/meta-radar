import { describe, expect, it } from "vitest";
import { cardImageUrl, withCardImageSize } from "./card-images";

describe("cardImageUrl", () => {
    it("defaults to MD, which is what the backend always returns", () => {
        expect(cardImageUrl("TWM/EN/TWM_095_R_EN_MD.png")).toBe(
            "https://cards.metaradar-tcg.com/pokemon/TWM/EN/TWM_095_R_EN_MD.png",
        );
    });

    it("rewrites the size suffix for a smaller asset", () => {
        expect(cardImageUrl("TWM/EN/TWM_095_R_EN_MD.png", "SM")).toBe(
            "https://cards.metaradar-tcg.com/pokemon/TWM/EN/TWM_095_R_EN_SM.png",
        );
        expect(cardImageUrl("TWM/EN/TWM_095_R_EN_MD.png", "XS")).toBe(
            "https://cards.metaradar-tcg.com/pokemon/TWM/EN/TWM_095_R_EN_XS.png",
        );
    });

    it("drops the suffix entirely for XL, the bare filename", () => {
        expect(cardImageUrl("TWM/EN/TWM_095_R_EN_MD.png", "XL")).toBe(
            "https://cards.metaradar-tcg.com/pokemon/TWM/EN/TWM_095_R_EN.png",
        );
    });
});

describe("withCardImageSize", () => {
    it("re-sizes an already-built URL without needing the raw path", () => {
        const mdUrl =
            "https://cards.metaradar-tcg.com/pokemon/TWM/EN/TWM_095_R_EN_MD.png";

        expect(withCardImageSize(mdUrl, "SM")).toBe(
            "https://cards.metaradar-tcg.com/pokemon/TWM/EN/TWM_095_R_EN_SM.png",
        );
        expect(withCardImageSize(mdUrl, "XS")).toBe(
            "https://cards.metaradar-tcg.com/pokemon/TWM/EN/TWM_095_R_EN_XS.png",
        );
        expect(withCardImageSize(mdUrl, "MD")).toBe(mdUrl);
    });
});

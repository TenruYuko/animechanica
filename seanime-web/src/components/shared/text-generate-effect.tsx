import { cn } from "@/components/ui/core/styling"
import { motion, stagger, useAnimate } from "framer-motion"
import React, { useEffect } from "react"

export const TextGenerateEffect = ({
    words,
    className,
    style,
    ...rest
}: {
    words: string;
    className?: string;
    style?: any
} & React.HTMLAttributes<HTMLDivElement>) => {
    const [scope, animate] = useAnimate()
    // Store the original words to prevent hydration errors
    const [displayWords, setDisplayWords] = React.useState<string>("");
    // Create a stable key for the entire component based on the words
    const contentKey = React.useMemo(() => (typeof words === 'string' ? words.replace(/\s+/g, "") : ""), [words])
    const wordsArray = displayWords.split(" ")

    // Handle hydration - only set displayWords after component mounts to ensure server/client match
    React.useEffect(() => {
        setDisplayWords(words || "S e a n i m e");
    }, [words]);

    // Reset and run animation when displayWords change (client-side only)
    useEffect(() => {
        // Skip animation if not mounted or no words
        if (!displayWords) return;
        
        // Use a safer approach to animation with proper error handling
        let isMounted = true;
        
        const resetAnimation = async () => {
            try {
                // Simple immediate fade-in without the two-step approach to avoid issues
                if (isMounted) {
                    animate(
                        "span",
                        { opacity: 1 },
                        {
                            duration: 1.5,
                            delay: stagger(0.15),
                        }
                    );
                }
            } catch (error) {
                console.error("Animation error:", error);
            }
        };
        
        // Small delay to ensure DOM is ready
        const timer = setTimeout(() => {
            if (isMounted) {
                resetAnimation();
            }
        }, 50);
        
        // Cleanup function
        return () => {
            isMounted = false;
            clearTimeout(timer);
        };
    }, [displayWords, animate]);

    const renderWords = () => {
        return (
            <motion.div ref={scope} key={contentKey}>
                {wordsArray.map((word, idx) => {
                    return (
                        <motion.span
                            key={`${contentKey}-${idx}`}
                            className="opacity-0"
                        >
                            {word}{" "}
                        </motion.span>
                    )
                })}
            </motion.div>
        )
    }

    return (
        <div className={cn("font-bold", className)} style={style} {...rest}>
            {renderWords()}
        </div>
    )
}

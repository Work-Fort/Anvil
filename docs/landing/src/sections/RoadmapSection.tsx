import { useRef, useLayoutEffect, useState, useEffect } from 'react';
import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import { Shield, Workflow, Award } from 'lucide-react';

gsap.registerPlugin(ScrollTrigger);

interface RoadmapSectionProps {
  className?: string;
}

interface Release {
  date: string;
  version: string;
  note: string;
  url: string;
}

const REPO_OWNER = 'get-virgil';
const REPO_NAME = 'cracker-barrel';

const stackItems = [
  {
    title: 'Certified Authentic',
    description: 'kernel.org autosigner PGP verification. Every kernel fully traceable from the pasture.',
    icon: Shield
  },
  {
    title: 'The Grill',
    description: 'Grade A5 Firecracker compatibility. For the most demanding config connoisseurs.',
    icon: Workflow
  },
  {
    title: 'Fresh Ingredients',
    description: 'Have your kernels like your coffee, hot and fresh every morning.',
    icon: Award
  }
];

export default function RoadmapSection({ className = '' }: RoadmapSectionProps) {
  const sectionRef = useRef<HTMLDivElement>(null);
  const headerRef = useRef<HTMLDivElement>(null);
  const cardsRef = useRef<(HTMLDivElement | null)[]>([]);

  const [releases, setReleases] = useState<Release[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  // Fetch recent releases from GitHub
  useEffect(() => {
    const fetchReleases = async () => {
      try {
        const response = await fetch(`https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/releases?per_page=4`);
        const data = await response.json();

        if (Array.isArray(data)) {
          const formattedReleases: Release[] = data.map((release: any) => ({
            date: new Date(release.published_at).toISOString().split('T')[0],
            version: release.tag_name,
            note: release.name || `Linux Kernel ${release.tag_name.replace('v', '')}`,
            url: release.html_url,
          }));

          setReleases(formattedReleases);
        }
      } catch (error) {
        console.error('Failed to fetch releases:', error);
        // Keep empty array on error
      } finally {
        setIsLoading(false);
      }
    };

    fetchReleases();
  }, []);

  useLayoutEffect(() => {
    const section = sectionRef.current;
    if (!section) return;

    const ctx = gsap.context(() => {
      // Header animation
      gsap.fromTo(headerRef.current,
        { y: 30, opacity: 0 },
        {
          y: 0,
          opacity: 1,
          duration: 0.6,
          ease: 'power2.out',
          scrollTrigger: {
            trigger: headerRef.current,
            start: 'top 80%',
            end: 'top 50%',
            scrub: true,
          }
        }
      );

      // Cards animation (staggered)
      cardsRef.current.forEach((card) => {
        if (!card) return;
        gsap.fromTo(card,
          { y: 60, opacity: 0 },
          {
            y: 0,
            opacity: 1,
            duration: 0.6,
            ease: 'power2.out',
            scrollTrigger: {
              trigger: card,
              start: 'top 75%',
              end: 'top 45%',
              scrub: true,
            }
          }
        );
      });

    }, section);

    return () => ctx.revert();
  }, []);

  return (
    <section 
      ref={sectionRef}
      className={`relative w-full min-h-screen py-20 sm:py-28 ${className}`}
    >
      {/* Background Image */}
      <div className="absolute inset-0 w-full h-full">
        <img 
          src="./roadmap_city_bg.jpg" 
          alt="Night City"
          className="w-full h-full object-cover"
        />
        <div className="absolute inset-0 bg-[#07070A]/85" />
      </div>

      {/* Content */}
      <div className="relative z-10 max-w-[980px] mx-auto px-6">
        {/* Header */}
        <div ref={headerRef} className="text-center mb-12 sm:mb-16">
          <span className="text-[#B9FF2C] text-sm font-semibold tracking-wider uppercase mb-3 block">
            The Stack
          </span>
          <h2 className="text-[clamp(32px,5vw,48px)] font-black text-[#F4F6FA] mb-4">
            Under the hood
          </h2>
          <p className="text-[#A7ACB8] text-base sm:text-lg max-w-xl mx-auto">
            Built on open source. Verified at every step. Automated end-to-end.
          </p>
        </div>

        {/* Stack Cards */}
        <div className="space-y-4 sm:space-y-6">
          {stackItems.map((item, index) => (
            <div
              key={item.title}
              ref={el => { cardsRef.current[index] = el; }}
              className="group relative bg-[#141419] border border-white/[0.08] rounded-[18px] p-6 sm:p-8 hover:border-[#B9FF2C]/30 transition-all duration-300 hover:-translate-y-1"
            >
              <div className="flex flex-col sm:flex-row sm:items-center gap-4 sm:gap-6">
                {/* Icon */}
                <div className="flex-shrink-0 w-12 h-12 rounded-xl bg-[#B9FF2C]/10 flex items-center justify-center group-hover:bg-[#B9FF2C]/20 transition-colors">
                  <item.icon className="w-6 h-6 text-[#B9FF2C]" />
                </div>

                {/* Content */}
                <div className="flex-1">
                  <div className="flex flex-wrap items-center gap-2 mb-2">
                    <span className="text-[#F4F6FA] font-bold text-lg">
                      {item.title}
                    </span>
                  </div>
                  <p className="text-[#A7ACB8] text-sm sm:text-base">
                    {item.description}
                  </p>
                </div>
              </div>
            </div>
          ))}
        </div>

        {/* Release Notes */}
        <div className="mt-12 sm:mt-16">
          <h3 className="text-[#F4F6FA] font-bold text-lg mb-6">Recent Releases</h3>
          <div className="space-y-3">
            {isLoading ? (
              <div className="text-center py-8">
                <p className="text-[#6F7682] text-sm">Loading releases...</p>
              </div>
            ) : releases.length > 0 ? (
              releases.map((release, i) => (
                <a
                  key={i}
                  href={release.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="flex flex-wrap items-center gap-3 sm:gap-4 text-sm py-2 border-b border-white/[0.06] last:border-0 hover:bg-white/[0.02] transition-colors -mx-2 px-2 rounded"
                >
                  <span className="text-[#6F7682] font-mono">{release.date}</span>
                  <span className="text-[#B9FF2C] font-mono">{release.version}</span>
                  <span className="text-[#A7ACB8]">{release.note}</span>
                </a>
              ))
            ) : (
              <div className="text-center py-8">
                <p className="text-[#A7ACB8] text-sm mb-2">No releases yet. First kernels coming soon!</p>
                <p className="text-[#6F7682] text-xs">Check back after the signing key is generated.</p>
              </div>
            )}
          </div>
        </div>
      </div>
    </section>
  );
}

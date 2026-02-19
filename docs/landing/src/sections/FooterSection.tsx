import { useRef, useLayoutEffect } from 'react';
import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import { BookOpen, Github, FileText, Shield } from 'lucide-react';

gsap.registerPlugin(ScrollTrigger);

interface FooterSectionProps {
  className?: string;
}

const REPO_OWNER = 'get-virgil';
const REPO_NAME = 'cracker-barrel';

const footerLinks = [
  { label: 'Docs', href: './docs/index.html', icon: BookOpen },
  { label: 'GitHub', href: `https://github.com/${REPO_OWNER}/${REPO_NAME}`, icon: Github },
  { label: 'Security Model', href: './docs/reference/security-model.html', icon: Shield },
  { label: 'Apache 2.0', href: `https://github.com/${REPO_OWNER}/${REPO_NAME}/blob/master/LICENSE.md`, icon: FileText },
];

export default function FooterSection({ className = '' }: FooterSectionProps) {
  const sectionRef = useRef<HTMLDivElement>(null);
  const contentRef = useRef<HTMLDivElement>(null);
  const ctaRef = useRef<HTMLAnchorElement>(null);
  const linksRef = useRef<HTMLDivElement>(null);

  useLayoutEffect(() => {
    const section = sectionRef.current;
    if (!section) return;

    const ctx = gsap.context(() => {
      // Content animation
      gsap.fromTo(contentRef.current,
        { y: 30, opacity: 0 },
        {
          y: 0,
          opacity: 1,
          duration: 0.6,
          ease: 'power2.out',
          scrollTrigger: {
            trigger: contentRef.current,
            start: 'top 80%',
            end: 'top 50%',
            scrub: true,
          }
        }
      );

      // CTA button animation
      gsap.fromTo(ctaRef.current,
        { y: 20, opacity: 0 },
        {
          y: 0,
          opacity: 1,
          duration: 0.5,
          ease: 'power2.out',
          scrollTrigger: {
            trigger: ctaRef.current,
            start: 'top 80%',
            end: 'top 60%',
            scrub: true,
          }
        }
      );

      // Links animation
      gsap.fromTo(linksRef.current,
        { opacity: 0 },
        {
          opacity: 1,
          duration: 0.5,
          ease: 'power2.out',
          scrollTrigger: {
            trigger: linksRef.current,
            start: 'top 90%',
            end: 'top 70%',
            scrub: true,
          }
        }
      );

    }, section);

    return () => ctx.revert();
  }, []);

  return (
    <section 
      ref={sectionRef}
      id="install"
      className={`relative w-full min-h-screen py-20 sm:py-28 ${className}`}
    >
      {/* Background Image */}
      <div className="absolute inset-0 w-full h-full">
        <img 
          src="./footer_diner_bg.jpg" 
          alt="Diner Interior"
          className="w-full h-full object-cover"
        />
        <div className="absolute inset-0 bg-gradient-to-t from-[#07070A] via-[#07070A]/80 to-[#07070A]/60" />
      </div>

      {/* Content */}
      <div className="relative z-10 max-w-[920px] mx-auto px-6 flex flex-col items-center justify-center min-h-[60vh]">
        {/* Main Content */}
        <div ref={contentRef} className="text-center mb-10 sm:mb-12">
          <h2 className="text-[clamp(36px,6vw,56px)] font-black text-[#F4F6FA] mb-4">
            Check, please.
          </h2>
          <p className="text-[#A7ACB8] text-base sm:text-lg max-w-md mx-auto">
            Let us wrap those up for you, you can take them to go.
          </p>
        </div>

        {/* CTA Button */}
        <a
          ref={ctaRef}
          href={`https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/latest`}
          target="_blank"
          rel="noopener noreferrer"
          className="inline-flex items-center gap-2 bg-[#B9FF2C] text-[#07070A] px-8 py-4 rounded-full text-lg font-bold hover:brightness-110 transition-all hover:-translate-y-0.5 hover:neon-box-glow mb-16 sm:mb-20"
        >
          <Github className="w-6 h-6" />
          View Latest Release on GitHub
        </a>

        {/* Footer Links */}
        <div 
          ref={linksRef}
          className="flex flex-wrap items-center justify-center gap-6 sm:gap-8"
        >
          {footerLinks.map((link) => (
            <a
              key={link.label}
              href={link.href}
              className="flex items-center gap-2 text-[#A7ACB8] hover:text-[#B9FF2C] transition-colors text-sm"
            >
              <link.icon className="w-4 h-4" />
              {link.label}
            </a>
          ))}
        </div>

        {/* Copyright */}
        <div className="mt-12 sm:mt-16 text-center">
          <p className="text-[#6F7682] text-xs">
            © 2026 Kaz Walker. Apache 2.0 License.
          </p>
        </div>
      </div>
    </section>
  );
}
